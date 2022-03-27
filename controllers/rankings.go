package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const ROUTINE_TIME = 24 * time.Hour // EVERY DAY WE DO THIS BECAUSE IT OWNS THE OONI API

type Country struct {
	Alpha_2 string
	Count   int
	Name    string
}
type BlockedWebsite struct {
	CountryName   string
	Website       string
	Blocked       bool
	LastUpdatedAt int64
}
type CountriesResponse struct {
	Countries []Country
}
type WebsiteNetworksResponse struct {
	Results []map[string]interface{}
}
type CountryScore struct {
	CountryName   string
	Score         int
	Ranking       int
	CommonBlocked map[string]bool
}
type CountryNoBlockedScore struct {
	CountryName string
	Score       int
	Ranking     int
}
type WebsiteNetwork struct {
	Count     int
	Probe_asn int
}
type WebsiteStat struct {
	Anomaly_count   int
	Confirmed_count int
	Failure_count   int
	Test_day        string
	Total_count     int
}
type WebsiteStatsResponse struct {
	Results []WebsiteStat
}

// TODO: Exponential Backoff with Circuit Breaker pattern
func fetchCountries() CountriesResponse {
	log.Println("Fetching Countries")
	resp, err := http.Get("https://api.ooni.io/api/_/countries")
	if err != nil {
		log.Println("Failed to read countries from ooni api", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read countries body", err)
	}
	var countries CountriesResponse
	json.Unmarshal(body, &countries)
	log.Println(countries)
	return countries
}

// TODO switch to measurements API: https://api.ooni.io/api/v1/measurements?limit=50&failure=false&probe_cc=RU&domain=https:%2F%2Fwww.youtube.com%2F&probe_asn=12389&test_name=web_connectivity&since=2022-02-25&until=2022-03-28
// it is a better endpoint zzz
func processWebsite(wsite string, country_cc string, asn float64, attempt int) bool {
	tmpURL := url.URL{
		Scheme: "https",
		Host:   "api.ooni.io",
		Path:   "api/_/website_stats",
	}
	q := tmpURL.Query()
	q.Add("probe_cc", country_cc)
	q.Add("probe_asn", strconv.Itoa(int(asn)))
	q.Add("input", wsite)
	tmpURL.RawQuery = q.Encode()
	resp, err := http.Get(tmpURL.String())
	if err != nil {
		log.Println("failed to process website ", wsite, err)
		if attempt != 15 { // giveup after 15 attempts.
			max := math.Min(1000, 5*math.Pow(2, float64(attempt)))
			r := math.Floor(rand.Float64() * (max))
			log.Printf("Attempting website: %s for country: %s again in %v, attempt: %d\n", wsite, country_cc, r, attempt)
			time.Sleep(time.Duration(r))
			return processWebsite(wsite, country_cc, asn, attempt+1)
		}
		log.Printf("failed to process website: %s for country: %s in 15 attempts, just going to say it is not blocked\n", wsite, country_cc)
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read country body ", err)
	}
	var tmpStruct3 map[string]interface{}
	json.Unmarshal(body, &tmpStruct3)
	average := 0.0
	anomaly_average := 0.0
	temp, ok := tmpStruct3["results"].([]interface{})
	if !ok {
		return false
	}
	for _, stat := range temp {
		average += (float64(stat.(map[string]interface{})["confirmed_count"].(float64)) / float64(stat.(map[string]interface{})["total_count"].(float64)))
		anomaly_average += (float64(stat.(map[string]interface{})["anomaly_count"].(float64)) / float64(stat.(map[string]interface{})["total_count"].(float64)))
	}
	average = (average / float64(len(tmpStruct3["results"].([]interface{})))) * 100
	anomaly_average = (anomaly_average / float64(len(tmpStruct3["results"].([]interface{})))) * 100
	is_blocked := false
	if average >= 50.0 || anomaly_average >= 70.0 { // 50% confidence its actually blocked
		is_blocked = true
	}
	return is_blocked
}

// TODO: Exponential Backoff with Circuit Breaker pattern
func processCountry(country Country, scores chan CountryScore, COMMON_WEBSITES []string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("error in processing country: %s, panic error: %v\n", country.Name, r)
		}
	}()
	log.Printf("Country Worker Started For Country: %s\n", country.Name)
	website_networks_url := fmt.Sprintf("https://api.ooni.io/api/_/website_networks?probe_cc=%s", country.Alpha_2)
	log.Println(website_networks_url)
	resp, err := http.Get(website_networks_url)
	if err != nil {
		log.Println("failed to process country ", country.Name, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read website_data body ", err)
	}
	var tmpStruct WebsiteNetworksResponse
	json.Unmarshal(body, &tmpStruct)
	results := tmpStruct.Results
	if len(results) == 0 {
		log.Printf("%s contains no data\n", country.Name)
		scores <- CountryScore{CountryName: country.Name, Score: 0}
		return
	}
	asn := results[0]["probe_asn"].(float64) // results[0].(WebsiteNetwork) // results[0].Probe_asn

	u := fmt.Sprintf("https://api.ooni.io/api/_/website_urls?limit=%s&offset=0&probe_cc=%s&probe_asn=%d" /* strconv.FormatUint(math.MaxUint64, 10) */, "10", country.Alpha_2, int(asn))
	log.Println(u)
	log.Println("attempting to fetch: ", u)
	resp, err = http.Get(u)
	if err != nil {
		log.Println("failed to process country ", country.Name, err)
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read country body ", err)
	}
	var tmpStruct2 map[string]interface{}
	json.Unmarshal(body, &tmpStruct2)
	score := tmpStruct2["metadata"].(map[string]interface{})["total_count"].(float64) * -1 // the more blocked websites the lower the score
	common_webites := map[string]bool{}
	for _, c := range COMMON_WEBSITES {
		is_blocked := processWebsite(fmt.Sprintf("http://%s/", strings.Trim(c, "\r")), country.Alpha_2, asn, 1)
		if !is_blocked {
			is_blocked = processWebsite(fmt.Sprintf("https://%s/", strings.Trim(c, "\r")), country.Alpha_2, asn, 1)
		}
		common_webites[strings.Trim(c, "\r")] = is_blocked
	}
	log.Printf("Country Worker Finished For Country: %s\n", country.Name)
	scores <- CountryScore{CountryName: country.Name, Score: int(score), CommonBlocked: common_webites}
}

// Every X Hours Recalculate Rankings
// And save to db
func RankingsRoutine() {
	log.Println("Reading common_websites from csv file")
	common_websites := strings.Split(readTopWebsitesCSV("static/topwebsites.csv"), "\n")
	start := time.Now()
	log.Println("Rankings Routine Begun")
	// Country Website Blockeds
	// https://api.ooni.io/api/_/website_urls?probe_cc=RU&probe_asn=12389
	countries := fetchCountries()
	scores := make(chan CountryScore)
	scoreArr := []CountryScore{}
	for _, country := range countries.Countries {
		go processCountry(country, scores, common_websites)
	}
	for range countries.Countries {
		log.Println("Waiting On Country Processing Routine...")
		score := <-scores
		scoreArr = append(scoreArr, score)
	}
	sort.Slice(scoreArr, func(i, j int) bool {
		return scoreArr[i].Score < scoreArr[j].Score
	})
	for i, score := range scoreArr {
		cpy := score
		cpy.Ranking = len(scoreArr) - i
		go func(score CountryScore, scores chan CountryScore) {
			_, err := AddScore(score)
			if err != nil {
				panic(err)
			}
			scores <- score
		}(cpy, scores)
	}
	for range scoreArr {
		log.Println("Waiting On Ranking/Add To Database...")
		<-scores
	}
	// Website Blocked Stats
	// https://api.ooni.io/api/v1/measurements?limit=50&failure=false&domain=www.linkedin.com&probe_asn=12389&test_name=web_connectivity&since=2022-02-18&until=2022-03-21
	time.AfterFunc(ROUTINE_TIME, RankingsRoutine)
	t := time.Now()
	elapsed := t.Sub(start)
	log.Printf("Rankings Routine Ended, It Took %s, Sleeping for %sms\n", elapsed.String(), ROUTINE_TIME.String())
}

func getRankings() echo.HandlerFunc {
	return func(c echo.Context) error {
		scores, err := GetScores()
		for i, _ := range scores {
			scores[i].CountryName = strings.Title(scores[i].CountryName)
		}
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, scores)
	}
}

func getRanking() echo.HandlerFunc {
	return func(c echo.Context) error {
		cName := c.Param("country")
		score, err := GetScore(cName)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, score)
	}
}

func Rankings(e *echo.Echo) {
	log.Println("🚀 /api/v1/countries/rankings - GET - Retrieve All Countries Ranked (Lower the number the worse)")
	log.Println("🚀 /api/v1/countries/{country: string} - GET - Retrieve Country Details")
	e.GET("/api/v1/countries/rankings", getRankings())
	e.GET("/api/v1/countries/rankings/:country", getRanking())
}

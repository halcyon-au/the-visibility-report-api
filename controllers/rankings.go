package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

const ROUTINE_TIME = 24 * time.Hour // EVERY DAY WE DO THIS BECAUSE IT OWNS THE OONI API

type WebsiteStatus = int8

const (
	Unblocked WebsiteStatus = 0
	Blocked   WebsiteStatus = 1
	Possible  WebsiteStatus = 2
)

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
	CountryName string
	Score       int
	Ranking     int
}
type CountryScoreWBlocked struct {
	CountryName       string
	Score             int
	Ranking           int
	BlockedWebsites   []string
	UnblockedWebsites []string
	PossibleWebsites  []string
	Websites          []string
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
type ProcessCountryChannelStruct struct {
	CountryScore      CountryScore
	BlockedWebsites   []string
	UnblockedWebsites []string
	PossibleWebsites  []string
	Websites          []string
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

func processWebsite(wSiteStruct map[string]interface{}) (WebsiteStatus, error) {
	if wSiteStruct["total_count"].(float64) < 20 {
		return Unblocked, errors.New("not enough data to be confident")
	}
	confirmed_percent := wSiteStruct["confirmed_count"].(float64) / wSiteStruct["total_count"].(float64)
	possible_percent := wSiteStruct["anomaly_count"].(float64) / wSiteStruct["total_count"].(float64)
	if confirmed_percent >= 0.5 {
		return Blocked, nil
	} else if wSiteStruct["anomaly_count"].(float64) > 0 && possible_percent >= 0.8 {
		return Possible, nil
	}
	return Unblocked, nil
}

// TODO: Exponential Backoff with Circuit Breaker pattern
func processCountry(country Country, scores chan ProcessCountryChannelStruct, COMMON_WEBSITES []string) {
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
		scores <- ProcessCountryChannelStruct{
			CountryScore:      CountryScore{CountryName: country.Name, Score: 0},
			BlockedWebsites:   []string{},
			UnblockedWebsites: []string{},
			PossibleWebsites:  []string{},
			Websites:          []string{},
		}
		return
	}
	asn := results[0]["probe_asn"].(float64) // results[0].(WebsiteNetwork) // results[0].Probe_asn

	u := fmt.Sprintf("https://api.ooni.io/api/_/website_urls?limit=%s&offset=0&probe_cc=%s&probe_asn=%d", strconv.FormatUint(math.MaxUint64, 10), country.Alpha_2, int(asn))
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
	blocked_websites := []string{}
	unblocked_websites := []string{}
	possible_websites := []string{}
	webs := []string{}
	for _, c := range tmpStruct2["results"].([]interface{}) {
		status, err := processWebsite(c.(map[string]interface{}))
		if err == nil {
			if status == Blocked {
				blocked_websites = append(blocked_websites, strings.ToLower(c.(map[string]interface{})["input"].(string)))
			} else if status == Possible {
				possible_websites = append(possible_websites, strings.ToLower(c.(map[string]interface{})["input"].(string)))
			} else {
				unblocked_websites = append(unblocked_websites, strings.ToLower(c.(map[string]interface{})["input"].(string)))
			}
		}
		webs = append(webs, strings.ToLower(c.(map[string]interface{})["input"].(string)))
	}
	log.Printf("Country Worker Finished For Country: %s\n", country.Name)
	scores <- ProcessCountryChannelStruct{
		CountryScore:      CountryScore{CountryName: country.Name, Score: len(blocked_websites) + len(possible_websites)},
		BlockedWebsites:   blocked_websites,
		UnblockedWebsites: unblocked_websites,
		PossibleWebsites:  possible_websites,
		Websites:          webs,
	}
}

// Every X Hours Recalculate Rankings
// And save to db
func RankingsRoutine(exitChannel chan os.Signal) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("I am panicing, as such I will now terminate the worker", r)
			signal.Notify(exitChannel, syscall.SIGINT, syscall.SIGTERM)
		}
	}()
	log.Println("Reading common_websites from csv file")
	common_websites := strings.Split(readTopWebsitesCSV("/app/static/topwebsites.csv"), "\n")
	start := time.Now()
	log.Println("Rankings Routine Begun")
	// Country Website Blockeds
	// https://api.ooni.io/api/_/website_urls?probe_cc=RU&probe_asn=12389
	countries := fetchCountries()
	scores := make(chan ProcessCountryChannelStruct)
	processArr := []ProcessCountryChannelStruct{}
	for _, country := range countries.Countries {
		go processCountry(country, scores, common_websites)
	}
	for range countries.Countries {
		log.Println("Waiting On Country Processing Routine...")
		process := <-scores
		processArr = append(processArr, process)
	}
	sort.Slice(processArr, func(i, j int) bool {
		return processArr[i].CountryScore.Score > processArr[j].CountryScore.Score
	})
	for i, process := range processArr {
		cpy := process
		cpy.CountryScore.Ranking = i + 1
		go func(process ProcessCountryChannelStruct, processes chan ProcessCountryChannelStruct) {
			_, err := AddProcess(process)
			if err != nil {
				panic(err)
			}
			processes <- process
		}(cpy, scores)
	}
	for range processArr {
		log.Println("Waiting On Ranking/Add To Database...")
		<-scores
	}
	// Website Blocked Stats
	// https://api.ooni.io/api/v1/measurements?limit=50&failure=false&domain=www.linkedin.com&probe_asn=12389&test_name=web_connectivity&since=2022-02-18&until=2022-03-21
	time.AfterFunc(ROUTINE_TIME, func() {
		RankingsRoutine(exitChannel)
	})
	t := time.Now()
	elapsed := t.Sub(start)
	log.Printf("Rankings Routine Ended, It Took %s, Sleeping for %sms\n", elapsed.String(), ROUTINE_TIME.String())
}

// GetRankings godoc
// @Summary  Retrieve All Countries Ranked (Lower the number the worse)
// @Tags     rankings
// @Produce  json
// @Success  200  {object}  []CountryScore
// @Failure  500  {object}  map[string]string
// @Router   /api/v1/countries/rankings [get]
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

// GetRankingByCountry godoc
// @Summary      Retrieve Country Details
// @Tags         rankings
// @Description  Get ranking details by country
// @Produce      json
// @Param        country  path      string  true  "Country Name"
// @Success      200      {object}  CountryScoreWBlocked
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/countries/{country} [get]
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

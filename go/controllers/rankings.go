package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const ROUTINE_TIME = 1 * time.Minute

type Country struct {
	Alpha_2 string
	Count   int
	Name    string
}
type BlockedWebsite struct {
	CountryName string
	Website     string
}
type CountriesResponse struct {
	Countries []Country
}
type WebsiteNetworksResponse struct {
	Results []map[string]interface{}
}
type CountryScore struct {
	CountryName     string
	Score           int
	Ranking         int
	BlockedWebsites []string
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

// TODO: Exponential Backoff with Circuit Breaker pattern
func processCountry(country Country, scores chan CountryScore) {
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

	url := fmt.Sprintf("https://api.ooni.io/api/_/website_urls?probe_cc=%s&probe_asn=%d", country.Alpha_2, int(asn))
	log.Println(url)
	log.Println("attempting to fetch: ", url)
	resp, err = http.Get(url)
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
	// TODO CRAWL "next_url": "https://api.ooni.io/api/_/website_urls?limit=10&offset=10&probe_asn=12389&probe_cc=RU", UNTIL EMPTY
	blocked_websites_struct := tmpStruct2["results"].([]interface{})
	blocked_websites := []string{}
	for _, strut := range blocked_websites_struct {
		blocked_websites = append(blocked_websites, strut.(map[string]interface{})["input"].(string))
	}
	log.Printf("Country Worker Finished For Country: %s\n", country.Name)
	scores <- CountryScore{CountryName: country.Name, Score: int(score), BlockedWebsites: blocked_websites}
}

// Every X Hours Recalculate Rankings
// And save to db
func RankingsRoutine() {
	start := time.Now()
	log.Println("Rankings Routine Begun")
	// Country Website Blockeds
	// https://api.ooni.io/api/_/website_urls?probe_cc=RU&probe_asn=12389
	countries := fetchCountries()
	scores := make(chan CountryScore)
	scoreArr := []CountryScore{}
	for _, country := range countries.Countries {
		go processCountry(country, scores)
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
			log.Println(score)
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

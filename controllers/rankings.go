package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const ROUTINE_TIME = 1 * time.Minute

type Country struct {
	Alpha_2 string
	Count   int
	Name    string
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
	log.Printf("Country Worker Finished For Country: %s\n", country.Name)
	scores <- CountryScore{CountryName: country.Name, Score: int(score)}
}

// Every X Hours Recalculate Rankings
// And save to db
func RankingsRoutine() {
	log.Println("Rankings Routine Begun")
	// Country Website Blockeds
	// https://api.ooni.io/api/_/website_urls?probe_cc=RU&probe_asn=12389
	countries := fetchCountries()
	scores := make(chan CountryScore)
	for _, country := range countries.Countries {
		go processCountry(country, scores)
	}
	for range countries.Countries {
		log.Println("Waiting On Country Processing Routine...")
		score := <-scores
		go func() {
			_, err := AddScore(score)
			if err != nil {
				panic(err)
			}
		}()
	}
	// Website Blocked Stats
	// https://api.ooni.io/api/v1/measurements?limit=50&failure=false&domain=www.linkedin.com&probe_asn=12389&test_name=web_connectivity&since=2022-02-18&until=2022-03-21
	time.AfterFunc(ROUTINE_TIME, RankingsRoutine)
	log.Printf("Rankings Routine Ended, Sleeping for %sms\n", ROUTINE_TIME.String())
}

package controllers

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/labstack/echo/v4"
)

type BlockStatus = uint8

const (
	Unblock BlockStatus = 0
	Block   BlockStatus = 1
	Possib  BlockStatus = 2
	Unknown BlockStatus = 3
)

func readTopWebsitesCSV(fileloc string) string {
	b, err := ioutil.ReadFile(fileloc)
	if err != nil {
		log.Println(err)
	}
	return string(b)
}

func BlockedWebsites(e *echo.Echo) {
	log.Println("🚀 /api/v1/blocked/{countryname - string}/{website - string} - GET - find closest block to website for countryname")
	log.Println("🚀 /api/v1/blocked/{countryname - string}/{website - string} - GET - find closest match to website for countryname, if there is match in blocked/unblocked return blocked/unblocked else return unknown")
	e.GET("/api/v1/blocked/:countryname/:website", getBlocked())
	e.GET("/api/v1/status/:countryname/:website", getStatus())
}

func StripWebsite(str string) string {
	return strings.ReplaceAll(strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(str, "https://", ""), "http://", "",
		), "/", "",
	), "www.", "")
}

func GetBlockedViaStripped(countryname string, website string) (string, float64, error) {
	fmt.Println(countryname)
	blocks, err := GetBlocks(countryname)
	fmt.Println(blocks)
	if err != nil {
		return "", 0.0, err
	}
	isBlocked := false
	matchedWith := ""
	simularity := 0.0
	for _, block := range blocks {
		strippedBlocked := StripWebsite(strings.ToLower(block))
		strippedInputCountry := StripWebsite(strings.ToLower(website))
		simularity = strutil.Similarity(strippedBlocked, strippedInputCountry, metrics.NewHamming())
		if simularity >= 0.4 {
			matchedWith = block
			isBlocked = true
			break
		}
	}
	if !isBlocked {
		simularity = 0.0
	}
	return matchedWith, simularity, nil
}

func checkArrForSimilar(searchArr []string, searchStr string, minMatch float64) (string, float64, bool) {
	found := false
	matchedWith := ""
	simularity := 0.0
	for _, str := range searchArr {
		strippedStr := StripWebsite(strings.ToLower(str))
		strippedSearch := StripWebsite(strings.ToLower(searchStr))
		simularity = strutil.Similarity(strippedStr, strippedSearch, metrics.NewHamming())
		if simularity >= minMatch {
			matchedWith = str
			found = true
			break
		}
	}
	return matchedWith, simularity, found
}

func getStatusViaStripped(countryname string, website string) (string, float64, uint8, error) {
	score, err := GetScore(countryname)
	matchedWith, simularityScore, found := checkArrForSimilar(score.BlockedWebsites, website, 0.75)
	if found {
		return matchedWith, simularityScore, Block, err
	}
	matchedWith, simularityScore, found = checkArrForSimilar(score.PossibleWebsites, website, 0.75)
	if found {
		return matchedWith, simularityScore, Possib, err
	}
	matchedWith, simularityScore, found = checkArrForSimilar(score.UnblockedWebsites, website, 0.75)
	if found {
		return matchedWith, simularityScore, Unblock, err
	}
	return matchedWith, simularityScore, Unknown, err
}

func getStatus() echo.HandlerFunc {
	return func(c echo.Context) error {
		country := c.Param("countryname")
		website := c.Param("website")
		matchedWith, simularity, status, err := getStatusViaStripped(country, website)
		if err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		switch status {
		case Block:
			return c.JSON(200, map[string]interface{}{"isBlocked": matchedWith != "", "matchedWith": matchedWith, "simularity": simularity, "status": "Blocked"})
		case Unblock:
			return c.JSON(200, map[string]interface{}{"isBlocked": matchedWith != "", "matchedWith": matchedWith, "simularity": simularity, "status": "Unblocked"})
		case Unknown:
			return c.JSON(200, map[string]interface{}{"isBlocked": matchedWith != "", "matchedWith": matchedWith, "simularity": simularity, "status": "Unknown"})
		case Possib:
			return c.JSON(200, map[string]interface{}{"isBlocked": matchedWith != "", "matchedWith": matchedWith, "simularity": simularity, "status": "Possible"})
		default:
			panic("that value should never happen")
		}
	}
}

// Using hamming simularity we find the closest similar website in blocked list
func getBlocked() echo.HandlerFunc {
	return func(c echo.Context) error {
		country := c.Param("countryname")
		website := c.Param("website")
		matchedWith, simularity, err := GetBlockedViaStripped(country, website)
		if err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		return c.JSON(200, map[string]interface{}{"isBlocked": matchedWith != "", "matchedWith": matchedWith, "simularity": simularity})
	}
}

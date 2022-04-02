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

func readTopWebsitesCSV(fileloc string) string {
	b, err := ioutil.ReadFile(fileloc)
	if err != nil {
		log.Println(err)
	}
	return string(b)
}

func BlockedWebsites(e *echo.Echo) {
	log.Println("🚀 /api/v1/blocked/{countryname - string}/{website - string} - GET - find closest block to website for countryname")
	e.GET("/api/v1/blocked/:countryname/:website", getBlocked())
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

// Using hamming simularity we find the closest similar website in blocked list
func getBlocked() echo.HandlerFunc {
	return func(c echo.Context) error {
		fmt.Println("asdf")
		country := c.Param("countryname")
		website := c.Param("website")
		matchedWith, simularity, err := GetBlockedViaStripped(country, website)
		if err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		return c.JSON(200, map[string]interface{}{"isBlocked": matchedWith != "", "matchedWith": matchedWith, "simularity": simularity})
	}
}

package controllers

import (
	"io/ioutil"
	"log"
	"strings"

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
	log.Println("🚀 /api/v1/blockedwebsites/{countryname - string} - GET - get all blocks for countryname")
	e.GET("/api/v1/blockedwebsites/:countryname", getBlocked())
}

func getBlocked() echo.HandlerFunc {
	return func(c echo.Context) error {
		countryName := c.Param("countryname")
		websites, err := GetBlocks(countryName)
		for i, _ := range websites {
			websites[i].CountryName = strings.Title(websites[i].CountryName)
		}
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, websites)
	}
}

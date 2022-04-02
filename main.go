package main

import (
	"os"
	"regexp"
	"strings"
	"visibilityreport/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var MODES = map[string]bool{ // Not sure if this is the best way to do this :/
	"local":       true,
	"development": true,
	"production":  true,
}

func regexOriginWrapper(regex string) func(string) (bool, error) {
	allowOrigin := func(origin string) (bool, error) {
		return regexp.MatchString(regex, origin)
	}
	return allowOrigin
}

func retrieveOrigins() func(string) (bool, error) {
	mode, found := os.LookupEnv("mode")
	if !found || mode == "" {
		panic("mode environment variable is not defined")
	}
	if _, exists := MODES[strings.ToLower(mode)]; !exists {
		panic("mode is not local, development or production")
	}
	switch mode { // ^https:\/\/labstack\.(net|com)$
	case "local":
		return regexOriginWrapper(`^http:\/\/localhost:[0-9]+$`)
	case "development":
		return regexOriginWrapper(`^https?:\/\/halycon.*$`)
	default:
		return regexOriginWrapper(`^https?:\/\/halycon.*$`)
	}
}

func main() {
	if err := controllers.IntializeDatabase(); err != nil {
		panic(err)
	}
	go controllers.RankingsRoutine()
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	originRegexFunc := retrieveOrigins()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOriginFunc: originRegexFunc,
		AllowHeaders:    []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	controllers.HeartBeat(e)
	controllers.Rankings(e)
	controllers.BlockedWebsites(e)
	e.Static("/", "static")
	e.Logger.Fatal(e.Start(":1323"))
}

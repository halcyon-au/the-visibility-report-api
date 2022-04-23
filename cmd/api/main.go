package main

import (
	"log"
	"visibilityreport/controllers"
	"visibilityreport/utils"

	_ "visibilityreport/cmd/api/docs"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title    The Visibility Report API
// @version  1.0
func main() {
	log.Println("Starting API")
	if err := controllers.IntializeDatabase(); err != nil {
		panic(err)
	}
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	originRegexFunc := utils.RetrieveOrigins()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOriginFunc: originRegexFunc,
		AllowHeaders:    []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	controllers.HeartBeat(e)
	controllers.Rankings(e)
	controllers.BlockedWebsites(e)

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.Static("/", "static")
	e.Logger.Fatal(e.Start(":1323"))
}

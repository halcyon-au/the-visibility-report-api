package main

import (
	"visibilityreport/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	if err := controllers.IntializeDatabase(); err != nil {
		panic(err)
	}
	go controllers.RankingsRoutine()
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	controllers.HeartBeat(e)
	controllers.Rankings(e)
	e.Logger.Fatal(e.Start(":1323"))
}

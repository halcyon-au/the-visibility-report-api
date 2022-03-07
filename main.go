package main

import (
	"visibilityreport/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	controllers.HeartBeat(e)
	e.Logger.Fatal(e.Start(":1323"))
}

package controllers

import (
	"log"

	"github.com/labstack/echo/v4"
)

func heartBeat() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "up and running 😎"})
	}
}

func HeartBeat(e *echo.Echo) {
	log.Println("🚀 /api/v1/hb - GET - Perform A Heartbeat")
	e.GET("/api/v1/hb", heartBeat())
}

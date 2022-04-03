package main

import (
	"log"
	"visibilityreport/controllers"
)

func main() {
	log.Println("Starting Worker...")
	if err := controllers.IntializeDatabase(); err != nil {
		panic(err)
	}
	controllers.RankingsRoutine()
	log.Fatalf("Worker Dying")
}

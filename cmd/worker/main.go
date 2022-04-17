package main

import (
	"log"
	"os"
	"visibilityreport/controllers"
)

func main() {
	log.Println("Starting Worker...")
	if err := controllers.IntializeDatabase(); err != nil {
		panic(err)
	}
	quitChannel := make(chan os.Signal, 1)
	go controllers.RankingsRoutine(quitChannel)
	<-quitChannel
	log.Fatalf("Worker Dying")
}

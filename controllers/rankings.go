package controllers

import (
	"log"
	"time"
)

const ROUTINE_TIME = 1 * time.Minute

// Every X Hours Recalculate Rankings
// And save to db
func RankingsRoutine() {
	log.Println("Rankings Routine Begun")

	time.AfterFunc(ROUTINE_TIME, RankingsRoutine)
	log.Printf("Rankings Routine Ended, Sleeping for %sms\n", ROUTINE_TIME.String())
}

package controllers

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var database *mongo.Database

func IntializeDatabase() error {
	username, found := os.LookupEnv("mongousername")
	if !found {
		return fmt.Errorf("username is not defined for mongousername env variable")
	}
	password, found := os.LookupEnv("mongopassword")
	if !found {
		return fmt.Errorf("password is not defined for mongopassword env variable")
	}
	hostname := "db"
	var err error
	client, err = mongo.NewClient(options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s", username, password, hostname)))
	if err != nil {
		return fmt.Errorf("failed to initialize mongo client - %s", err.Error())
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster - %s", err.Error())
	}
	// defer client.Disconnect(ctx) TODO: investigate how i can make it disconnect client once program is done
	database = client.Database("visiblityreport")
	return nil
}

func AddWebsiteBlock(country string, block string, done chan int) {
	blockCollection := database.Collection("blockedwebsites")
	opts := options.Replace().SetUpsert(true)
	_, err := blockCollection.ReplaceOne(context.TODO(), bson.M{
		"countryname": strings.ToLower(country),
		"website":     strings.ToLower(block),
	}, bson.D{
		{Key: "countryname", Value: strings.ToLower(country)},
		{Key: "website", Value: strings.ToLower(block)},
	}, opts)
	if err != nil {
		log.Printf("Failed to update website block %s for country %s\n", block, country)
	}
	done <- 1
}

// PUT operation i.e replace if it exists
func AddScore(cs CountryScore) (*mongo.UpdateResult, error) {
	log.Println("Adding to scores collection")
	doneChannel := make(chan int)
	scoreCollection := database.Collection("scores")
	opts := options.Replace().SetUpsert(true)
	insertResult, err := scoreCollection.ReplaceOne(context.TODO(), bson.M{
		"countryname": strings.ToLower(cs.CountryName),
	}, bson.D{
		{Key: "countryname", Value: strings.ToLower(cs.CountryName)},
		{Key: "score", Value: cs.Score},
		{Key: "ranking", Value: cs.Ranking},
	}, opts)
	if err != nil {
		log.Println(err)
		return insertResult, err
	}
	for _, block := range cs.BlockedWebsites {
		go AddWebsiteBlock(cs.CountryName, block, doneChannel)
	}
	for range cs.BlockedWebsites {
		log.Printf("waiting on insertion of website block for country %s\n", cs.CountryName)
		<-doneChannel
	}
	return insertResult, err
}

func GetScores() ([]CountryScore, error) {
	var results []CountryScore
	log.Println(database)
	scoreCollection := database.Collection("scores")
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "score", Value: 1}})
	cursor, err := scoreCollection.Find(context.TODO(), bson.D{}, opts)
	if err != nil {
		return results, fmt.Errorf("failed to retrieve scores - %s", err.Error())
	}
	if err = cursor.All(context.TODO(), &results); err != nil {
		return results, fmt.Errorf("failed to retrieve scores - %s", err.Error())
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Ranking > results[j].Ranking
	})
	return results, nil
}

func GetBlocks(countryname string) ([]BlockedWebsite, error) { // todo add field for recent so we can sort by how recent
	var results []BlockedWebsite
	log.Println(database)
	blockCollection := database.Collection("blockedwebsites")
	opts := options.Find()
	cursor, err := blockCollection.Find(context.TODO(), bson.D{{Key: "countryname", Value: strings.ToLower(countryname)}}, opts)
	if err != nil {
		return results, fmt.Errorf("failed to retrieve blocked websites - %s", err.Error())
	}
	if err = cursor.All(context.TODO(), &results); err != nil {
		return results, fmt.Errorf("failed to retrieve blocked websites - %s", err.Error())
	}
	log.Println(results)
	return results, nil
}

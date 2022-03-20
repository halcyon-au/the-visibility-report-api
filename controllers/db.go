package controllers

import (
	"context"
	"fmt"
	"os"
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

// PUT operation i.e replace if it exists
func AddScore(cs CountryScore) (*mongo.UpdateResult, error) {
	scoreCollection := database.Collection("scores")
	opts := options.Replace().SetUpsert(true)
	insertResult, err := scoreCollection.ReplaceOne(context.TODO(), bson.M{
		"countryname": cs.CountryName,
	}, bson.D{
		{Key: "countryname", Value: cs.CountryName},
		{Key: "score", Value: cs.Score},
	}, opts)
	return insertResult, err
}

func GetScores() error {
	fmt.Println(database)
	scoreCollection := database.Collection("scores")
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "score", Value: 1}})
	cursor, err := scoreCollection.Find(context.TODO(), bson.D{}, opts)
	if err != nil {
		return fmt.Errorf("failed to retrieve scores - %s", err.Error())
	}
	var results []CountryScore
	if err = cursor.All(context.TODO(), &results); err != nil {
		return fmt.Errorf("failed to retrieve scores - %s", err.Error())
	}
	for _, result := range results {
		fmt.Println(result)
	}
	return nil
}

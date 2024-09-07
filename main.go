package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Instance struct {
	InstanceID   string   `json:"instanceId"`
	Username     string   `json:"username"`
	BinaryWeeks  []string `json:"binaryWeeks"`
	CreationDate string   `json:"creationDate"`
}

type MongoRecord struct {
	InstanceID   string
	Username     string
	BinaryWeeks  string
	CreationDate string
}

func main() {
	fmt.Println("Starting server...")

	if os.Getenv("ENV") != "prod" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file", err)
		}
	}

	// Get environment variables
	dbURI := os.Getenv("DB_URI")
	dbName := os.Getenv("DB_NAME")
	dbCollectionName := os.Getenv("DB_COLLECTION")
	// MongoDB setup
	clientOptions := options.Client().ApplyURI(dbURI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	collection := client.Database(dbName).Collection(dbCollectionName)

	r := getRouterAndSetupMiddlewares()
	setRoutes(r, collection)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func IsBinaryString(s string) bool {
	for _, char := range s {
		if char != '0' && char != '1' {
			return false
		}
	}
	return true
}

func convertBinaryToDecimal(week string) (int64, error) {
	return strconv.ParseInt(week, 2, 8)
}

func convertDecimalWeekToBinary(binaryWeeks string) []string {
	weeks := strings.Split(binaryWeeks, "|")
	for i, week := range weeks {
		if num, err := strconv.ParseInt(week, 10, 8); err == nil {
			weeks[i] = fmt.Sprintf("%07b", num)
		}
	}
	return weeks
}

func getCreationDateByInstanceId(collection *mongo.Collection, id string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result MongoRecord
	err := collection.FindOne(ctx, bson.M{"instanceId": id}).Decode(&result)
	if err != nil {
		return ""
	}

	return result.CreationDate
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Instance struct {
	InstanceID  string   `json:"instanceId"`
	BinaryWeeks []string `json:"binaryWeeks"`
}

func main() {
	fmt.Println("Starting server...")
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file", err)
	// }

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

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// POST endpoint to store data with provided instanceId in MongoDB
	r.Post("/store", func(w http.ResponseWriter, r *http.Request) {
		var rv Instance
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&rv)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Insert document into MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = collection.InsertOne(ctx, bson.M{
			"instanceId":  rv.InstanceID,
			"binaryWeeks": rv.BinaryWeeks,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("Data stored successfully"))
	})

	// POST endpoint to generate a unique instanceId and store data in MongoDB
	r.Post("/generate", func(w http.ResponseWriter, r *http.Request) {
		var rv Instance
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&rv)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for _, week := range rv.BinaryWeeks {

			isBinaryWeek := IsBinaryString(week)
			if !isBinaryWeek || len(week) != 7 {
				http.Error(w, "Invalid data :(", http.StatusBadRequest)
				return
			}
		}

		// Generate a new UUID for instanceId
		rv.InstanceID = uuid.New().String()

		// Insert document into MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = collection.InsertOne(ctx, bson.M{
			"instanceId":  rv.InstanceID,
			"binaryWeeks": strings.Join(rv.BinaryWeeks, "|"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return the generated instanceId
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"instanceId": rv.InstanceID})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
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

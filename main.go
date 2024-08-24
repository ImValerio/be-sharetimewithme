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
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Instance struct {
	InstanceID  string   `json:"instanceId"`
	Username    string   `json:"username"`
	BinaryWeeks []string `json:"binaryWeeks"`
}

type MongoRecord struct {
	InstanceID  string
	Username    string
	BinaryWeeks string
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


	// POST endpoint to generate a unique instanceId and store data in MongoDB
	r.Post("/generate", func(w http.ResponseWriter, r *http.Request) {
		var rv Instance
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&rv)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for i, week := range rv.BinaryWeeks {

			isBinaryWeek := IsBinaryString(week)
			if !isBinaryWeek || len(week) != 7 {
				http.Error(w, "Invalid data :(", http.StatusBadRequest)
				return
			}
			decimalWeek, err:=  convertBinaryToDecimal(week)
			if err != nil {
				http.Error(w, "There was an issue during the conversion process :(", http.StatusInternalServerError)
				return
			}
			rv.BinaryWeeks[i] = strconv.FormatInt(decimalWeek, 10)
		}

		// Generate a new UUID for instanceId
		rv.InstanceID = uuid.New().String()

		// Insert document into MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = collection.InsertOne(ctx, bson.M{
			"instanceId":  rv.InstanceID,
			"username":    rv.Username,
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
	r.Post("/instance", func(w http.ResponseWriter, r *http.Request) {
			var rv Instance
			dec := json.NewDecoder(r.Body)
			err := dec.Decode(&rv)

			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if rv.InstanceID == "" || rv.Username == "" || len(rv.BinaryWeeks) == 0 {
				http.Error(w, "Missing required fields", http.StatusBadRequest)
				return
			}

			for i, week := range rv.BinaryWeeks {

				isBinaryWeek := IsBinaryString(week)
				if !isBinaryWeek || len(week) != 7 {
					http.Error(w, "Invalid data :(", http.StatusBadRequest)
					return
				}
 				decimalWeek, err := convertBinaryToDecimal(week)
				if err != nil {
					http.Error(w, "There was an issue during the conversion process :(", http.StatusInternalServerError)
					return
				}

				rv.BinaryWeeks[i] = strconv.FormatInt(decimalWeek, 10)
			}


			// Insert document into MongoDB
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err = collection.InsertOne(ctx, bson.M{
				"instanceId":  rv.InstanceID,
				"username":    rv.Username,
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

	r.Get("/instance/{id}", func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "id")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Insert document into MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := collection.Find(ctx, bson.M{"instanceId": id})

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer cursor.Close(ctx)

		var records []MongoRecord
		if err = cursor.All(ctx, &records); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var instances []Instance
		for _, data := range records {
			instances = append(instances, Instance{
				InstanceID:  data.InstanceID,
				Username:    data.Username,
				BinaryWeeks: convertDecimalWeekToBinary(data.BinaryWeeks),
			})
		}

		json.NewEncoder(w).Encode(instances)
	})

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
func convertBinaryToDecimal(week string) (int64,error) {
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
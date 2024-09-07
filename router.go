package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getRouterAndSetupMiddlewares() *chi.Mux {

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

	return r
}

func setRoutes(r *chi.Mux, collection *mongo.Collection) {

	r.Post("/instance", func(w http.ResponseWriter, r *http.Request) {
		var rv Instance
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&rv)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if rv.Username == "" || len(rv.BinaryWeeks) == 0 {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		if rv.InstanceID == "" {
			rv.InstanceID = uuid.New().String()
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

		// Check if the username already exists for the specific instanceId
		count, err := collection.CountDocuments(ctx, bson.M{"instanceId": rv.InstanceID, "username": rv.Username})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "Username already exists for this instance", http.StatusBadRequest)
			return
		}

		creationDate := time.Now().Format("2006/01/02")

		if rv.InstanceID != "" {
			creationDate = getCreationDateByInstanceId(collection, rv.InstanceID)
		}

		_, err = collection.InsertOne(ctx, bson.M{
			"instanceId":   rv.InstanceID,
			"username":     rv.Username,
			"binaryWeeks":  strings.Join(rv.BinaryWeeks, "|"),
			"creationDate": creationDate,
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
				InstanceID:   data.InstanceID,
				Username:     data.Username,
				BinaryWeeks:  convertDecimalWeekToBinary(data.BinaryWeeks),
				CreationDate: data.CreationDate,
			})
		}

		json.NewEncoder(w).Encode(instances)
	})
	r.Delete("/instance/{id}/{username}", func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "id")
		username := chi.URLParam(r, "username")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		filter := bson.M{"instanceId": id, "username": username}
		result, err := collection.DeleteOne(ctx, filter)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if result.DeletedCount == 0 {
			http.Error(w, "No records found to delete", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Record deleted successfully"})
	})

}

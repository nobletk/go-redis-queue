package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Result struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Status  string `json:"status"`
	Output  string `json:"output"`
}

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	http.HandleFunc("/results/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/results/"):]
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		data, err := rdb.HGet(ctx, "results", id).Result()
		if err == redis.Nil {
			http.Error(w, "not ready", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, data)
	})

	log.Println("Results Api running on :8082")
	http.ListenAndServe(":8082", nil)
}

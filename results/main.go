package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

const (
	TasksHash = "tasks"
)

type Task struct {
	ID          string `json:"id"`
	Message     string `json:"message"`
	CreatedAt   string `json:"created_at"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Status      string `json:"status"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	Retries     int    `json:"retries,omitempty"`
}

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	http.HandleFunc("/results/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/results/")
		if id == "" {
			http.Error(w, `{"error": "missing id"}`, http.StatusBadRequest)
			return
		}

		data, err := rdb.HGet(ctx, TasksHash, id).Result()
		if err == redis.Nil {
			http.Error(w, `{"error": "task not found"}`, http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err), http.StatusInternalServerError)
			return
		}

		var task Task
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "invalid task data: %s"}`, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if task.Status == "pending" || task.Status == "processing" {
			w.WriteHeader(http.StatusAccepted)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(task)
	})

	log.Println("Results Api running on :8082")
	http.ListenAndServe(":8082", nil)
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

const (
	TasksHash   = "tasks"
	EventsQueue = "events"
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

type SendRequest struct {
	Message string `json:"message"`
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

	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req SendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "invalid JSON: %s"}`, err), http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, `{"error": "missing message"}`, http.StatusBadRequest)
			return
		}

		task := Task{
			ID:        uuid.New().String(),
			Message:   req.Message,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Status:    "pending",
		}

		data, err := json.Marshal(task)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"`, err), http.StatusInternalServerError)
			return
		}

		pipe := rdb.TxPipeline()
		pipe.HSet(ctx, TasksHash, task.ID, string(data))
		pipe.LPush(ctx, EventsQueue, task.ID)
		if _, err := pipe.Exec(ctx); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": task.ID, "status": task.Status})
	})

	log.Println("Producer running on :8080")
	http.ListenAndServe(":8080", nil)
}

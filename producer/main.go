package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Event struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		msg := r.URL.Query().Get("msg")
		if msg == "" {
			http.Error(w, "missing msg param", http.StatusBadRequest)
			return
		}

		ev := Event{
			ID:      uuid.New().String(),
			Message: msg,
			Time:    "",
		}

		data, _ := json.Marshal(ev)

		if err := rdb.LPush(ctx, "events", data).Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "event_id=%s\n", ev.ID)
	})

	log.Println("Producer running on :8080")
	http.ListenAndServe(":8080", nil)
}

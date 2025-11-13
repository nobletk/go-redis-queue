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

func main() {
	redisAddr := os.Getenv("REDID_ADDR")
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
		if err := rdb.LPush(ctx, "events", msg).Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Queued message: %s\n", msg)
	})

	log.Println("Producer running on :8080")
	http.ListenAndServe(":8080", nil)
}

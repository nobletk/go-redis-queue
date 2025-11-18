package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Event struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

type Result struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Status  string `json:"status"`
	Output  string `json:"output"`
}

var ctx = context.Background()

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	log.Println("Worker started, waiting for events...")

	for {
		msg, err := rdb.BRPop(ctx, 0*time.Second, "events").Result()
		if err != nil {
			log.Println("BRPOP error:", err)
			time.Sleep(time.Second)
			continue
		}

		var ev Event
		json.Unmarshal([]byte(msg[1]), &ev)

		// Simulate heavy work
		time.Sleep(30 * time.Second)

		output := "Processed: " + ev.Message

		res := Result{
			ID:      ev.ID,
			Message: ev.Message,
			Status:  "done",
			Output:  output,
		}

		b, _ := json.Marshal(res)

		rdb.HSet(ctx, "results", ev.ID, b)

		log.Printf("Processed event %s -> stored result\n", ev.ID)
	}
}

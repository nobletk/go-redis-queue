package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	log.Println("Worker started, waiting for events...")

	for {
		msg, err := rdb.BRPop(ctx, 0*time.Second, "events").Result()
		if err != nil {
			log.Printf("Error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("Processed event: %s\n", msg[1])
	}
}

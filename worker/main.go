package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

const (
	TasksHash   = "tasks"
	EventsQueue = "events"
	DLQQueue    = "dlq"
	MaxRetries  = 3
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

	log.Println("Worker started, waiting for tasks...")

	for {
		msg, err := rdb.BLPop(ctx, 0, EventsQueue).Result()
		if err != nil {
			log.Printf("BLPOP error: %v", err)
			time.Sleep(time.Second * 2)
			continue
		}
		id := msg[1]

		rawData, err := rdb.HGet(ctx, TasksHash, id).Result()
		if err != nil {
			log.Printf("HGet error for %s: %v", id, err)
			continue
		}

		var task Task
		if err := json.Unmarshal([]byte(rawData), &task); err != nil {
			log.Printf("Unmarshal error for %s: %v", id, err)
			continue
		}

		task.Status = "processing"
		task.StartedAt = time.Now().UTC().Format(time.RFC3339)
		taskData, err := json.Marshal(task)
		if err != nil {
			log.Printf("Marshal error for %s: %v", id, err)
			continue
		}
		if err := rdb.HSet(ctx, TasksHash, id, string(taskData)).Err(); err != nil {
			log.Printf("HSet processing error for %s: %v", id, err)
			continue
		}

		// Simulate work
		time.Sleep(30 * time.Second)

		// Simulate random success or failure
		if rand.Intn(2) == 0 {
			task.Output = "Processed: " + task.Message
			task.Status = "done"
			task.Error = ""
			task.CompletedAt = time.Now().UTC().Format(time.RFC3339)
		} else {
			task.Status = "failed"
			task.Error = "Simulated processing failure"
			task.Retries++
		}

		if task.Status == "failed" {
			if task.Retries < MaxRetries {
				task.Status = "pending"
				log.Printf("Task %s failed, retrying (%d/%d)", id, task.Retries, MaxRetries)
			} else {
				if err := rdb.LPush(ctx, DLQQueue, id).Err(); err != nil {
					log.Printf("DLQ push error for %s: %v", id, err)
				}
				log.Printf("Task %s failed after %d retries, sent to DLQ", id, task.Retries)
			}
		}

		taskData, err = json.Marshal(task)
		if err != nil {
			log.Printf("Marshal error for %s: %v", id, err)
			task.Status = "failed"
			task.Error = "Marshal failed after processing"
			task.CompletedAt = time.Now().UTC().Format(time.RFC3339)
			taskData, _ = json.Marshal(task)
		}

		if err := rdb.HSet(ctx, TasksHash, id, string(taskData)).Err(); err != nil {
			log.Printf("HSet update error for %s: %v", id, err)
		} else {
			log.Printf("Updated task %s with status %s", id, task.Status)
		}

		if task.Status == "pending" {
			if err := rdb.LPush(ctx, EventsQueue, id).Err(); err != nil {
				log.Printf("Retry push error for %s: %v", id, err)
			}
		}
	}
}

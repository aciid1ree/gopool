package main

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Workers   int
	QueueSize int
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func LoadConfig() Config {
	workersStr := getEnv("WORKERS", "4")
	queueStr := getEnv("QUEUE_SIZE", "64")

	workers, err := strconv.Atoi(workersStr)
	if err != nil || workers <= 0 {
		log.Printf("Invalid worker count in env variable WORKERS, use 4")
		workers = 4
	}

	queueSize, err := strconv.Atoi(queueStr)
	if err != nil || queueSize <= 0 {
		log.Printf("Invalid queue size in env variable WORKERS, use 64")
		queueSize = 64
	}

	return Config{workers, queueSize}
}

func main() {
	cfg := LoadConfig()
	log.Printf("Starting service: %d workers, queue for %d tasks", cfg.Workers, cfg.QueueSize)
}

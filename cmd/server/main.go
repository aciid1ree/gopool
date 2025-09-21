package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"gopool/internal/httpapi"
	"gopool/internal/queue"
	"gopool/internal/worker"
)

const (
	defaultAddr      = ":8080"
	defaultWorkers   = 4
	defaultQueueSize = 64
)

func main() {
	addr := envStr("ADDR", defaultAddr)
	workers := envInt("WORKERS", defaultWorkers)
	queueSize := envInt("QUEUE_SIZE", defaultQueueSize)

	log.Printf("starting queue-svc: addr=%s WORKERS=%d QUEUE_SIZE=%d", addr, workers, queueSize)

	store := queue.NewStore()
	q := queue.New(queueSize, store)

	api := httpapi.New(q, store)
	mux := api.NewMux()
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	baseRNG := rand.New(rand.NewSource(time.Now().UnixNano()))

	pool := worker.NewPool(
		worker.WithWorkers(workers),
		worker.WithBackoffParams(50*time.Millisecond, 2*time.Second),
		worker.WithRNG(baseRNG),
		worker.WithRunner(func(t queue.Task) error {
			ms := 100 + rand.Intn(401) // 100..500
			time.Sleep(time.Duration(ms) * time.Millisecond)
			if rand.Intn(100) < 20 { // ~20% фейл
				return worker.ErrSimulated
			}
			return nil
		}),
	)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workersDone := make(chan struct{})
	go func() {
		pool.Run(rootCtx, q, store)
		close(workersDone)
	}()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("http server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case <-rootCtx.Done():
		log.Printf("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			log.Printf("server error: %v", err)
		}
	}

	q.Close()
	log.Printf("queue closed: rejecting new enqueues, draining buffered tasks")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		log.Printf("http shutdown error: %v", err)
	} else {
		log.Printf("http server shutdown complete")
	}

	select {
	case <-workersDone:
		log.Printf("workers drained and stopped")
	case <-shutdownCtx.Done():
		log.Printf("timeout waiting for workers to stop")
	}

	log.Printf("bye")
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil || i < 0 {
		return def
	}
	return i
}

func envStr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

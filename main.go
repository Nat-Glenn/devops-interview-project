package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if runHealthCheck() {
			return
		}
		os.Exit(1)
	}

	port := getEnv("PORT", "8080")
	store := NewMemoryStore()
	RegisterTaskMetrics(store)

	mux := http.NewServeMux()

	// Health check — must respond 200 for liveness probes
	mux.HandleFunc("GET /healthz", HealthHandler)

	// Metrics — Prometheus-compatible plaintext endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	// Task CRUD
	mux.HandleFunc("GET /tasks", ListTasksHandler(store))
	mux.HandleFunc("POST /tasks", CreateTaskHandler(store))
	mux.HandleFunc("GET /tasks/{id}", GetTaskHandler(store))
	mux.HandleFunc("PUT /tasks/{id}", UpdateTaskHandler(store))
	mux.HandleFunc("DELETE /tasks/{id}", DeleteTaskHandler(store))

	addr := ":" + port
	log.Printf("task-api starting on %s", addr)
	if err := http.ListenAndServe(addr, MetricsMiddleware(mux)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runHealthCheck() bool {
	port := getEnv("PORT", "8080")

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

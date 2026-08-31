package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_api_http_requests_total",
			Help: "Total number of HTTP requests handled by the task API.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "task_api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		route := r.Pattern
		if route == "" {
			route = "unknown"
		} else {
			if _, pattern, found := strings.Cut(route, " "); found {
				route = pattern
			}
		}

		httpRequestsTotal.WithLabelValues(
			r.Method,
			route,
			strconv.Itoa(recorder.status),
		).Inc()

		httpRequestDuration.WithLabelValues(
			r.Method,
			route,
		).Observe(time.Since(start).Seconds())
	})
}

func RegisterTaskMetrics(store Store) {
	prometheus.MustRegister(
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "task_api_tasks_total",
				Help: "Total number of tasks.",
			},
			func() float64 {
				total, _ := store.Stats()
				return float64(total)
			},
		),
	)

	prometheus.MustRegister(
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "task_api_tasks_done",
				Help: "Number of completed tasks.",
			},
			func() float64 {
				_, done := store.Stats()
				return float64(done)
			},
		),
	)
}

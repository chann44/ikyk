package internals

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsCollector struct {
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	errorCount      *prometheus.CounterVec
	cacheHits       *prometheus.CounterVec
	activeRequests  *prometheus.GaugeVec
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requestCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_requests_total",
				Help: "Total number of requests",
			},
			[]string{"service", "method", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gateway_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method"},
		),
		errorCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_errors_total",
				Help: "Total number of errors",
			},
			[]string{"service", "type"},
		),
		cacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"service"},
		),
		activeRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_active_requests",
				Help: "Number of active requests",
			},
			[]string{"service"},
		),
	}
}

func (mc *MetricsCollector) RecordRequest(service, method string, statusCode int, duration time.Duration) {
	mc.requestCount.WithLabelValues(service, method, strconv.Itoa(statusCode)).Inc()
	mc.requestDuration.WithLabelValues(service, method).Observe(duration.Seconds())
}

func (mc *MetricsCollector) RecordError(service, errorType string) {
	mc.errorCount.WithLabelValues(service, errorType).Inc()
}

func (mc *MetricsCollector) RecordCacheHit(service string) {
	mc.cacheHits.WithLabelValues(service).Inc()
}

func (mc *MetricsCollector) IncrementActive(service string) {
	mc.activeRequests.WithLabelValues(service).Inc()
}

func (mc *MetricsCollector) DecrementActive(service string) {
	mc.activeRequests.WithLabelValues(service).Dec()
}

func (mc *MetricsCollector) Handler() http.Handler {
	return promhttp.Handler()
}

// Middleware to track metrics
func (mc *MetricsCollector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		mc.RecordRequest(r.URL.Path, r.Method, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	httpActiveRequests  metric.Int64UpDownCounter
	appStartedAt        metric.Int64Gauge
)

type MetricsRecorder struct {
	meter metric.Meter
}

func NewMetricsRecorder(serviceName string) *MetricsRecorder {
	meter := otel.Meter(serviceName + "/metrics")
	return &MetricsRecorder{meter: meter}
}

func InitCustomMetrics(serviceName string) error {
	meter := otel.Meter(serviceName + "/metrics")

	var err error

	httpRequestsTotal, err = meter.Int64Counter(
		"app.http.requests.total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	httpRequestDuration, err = meter.Float64Histogram(
		"app.http.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return err
	}

	httpActiveRequests, err = meter.Int64UpDownCounter(
		"app.http.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	appStartedAt, err = meter.Int64Gauge(
		"app.started_at",
		metric.WithDescription("Application start timestamp (Unix)"),
		metric.WithUnit("{timestamp}"),
	)
	if err != nil {
		return err
	}

	appStartedAt.Record(context.Background(), time.Now().Unix())

	return nil
}

func RecordRequestMetrics(c *fiber.Ctx, duration time.Duration) {
	attrs := metric.WithAttributes(
		attribute.String("http.method", c.Method()),
		attribute.String("http.path", c.Route().Path),
		attribute.String("http.route", c.Route().Path),
		attribute.Int("http.status_code", c.Response().StatusCode()),
	)

	httpRequestsTotal.Add(context.Background(), 1, attrs)
	httpRequestDuration.Record(context.Background(), duration.Seconds(), attrs)
}

func IncActiveRequests(ctx context.Context) {
	if httpActiveRequests != nil {
		httpActiveRequests.Add(ctx, 1)
	}
}

func DecActiveRequests(ctx context.Context) {
	if httpActiveRequests != nil {
		httpActiveRequests.Add(ctx, -1)
	}
}

type MetricsMiddleware struct{}

func NewMetricsMiddleware() MetricsMiddleware {
	return MetricsMiddleware{}
}

func (m MetricsMiddleware) Handle(c *fiber.Ctx) error {
	start := time.Now()

	IncActiveRequests(c.UserContext())
	defer func() {
		DecActiveRequests(c.UserContext())
		RecordRequestMetrics(c, time.Since(start))
	}()

	return c.Next()
}

type BusinessMetrics struct {
	meter metric.Meter
}

func NewBusinessMetrics(serviceName string) (*BusinessMetrics, error) {
	meter := otel.Meter(serviceName + "/business")

	return &BusinessMetrics{meter: meter}, nil
}

func (m *BusinessMetrics) RecordListEvents(ctx context.Context, duration time.Duration, count int) {
	histogram, _ := m.meter.Float64Histogram(
		"app.events.list.duration",
		metric.WithDescription("List events query duration"),
		metric.WithUnit("s"),
	)
	histogram.Record(ctx, duration.Seconds())

	counter, _ := m.meter.Int64Counter(
		"app.events.list.total",
		metric.WithDescription("Total list events requests"),
		metric.WithUnit("{request}"),
	)
	counter.Add(ctx, 1)
}

func (m *BusinessMetrics) RecordGetEvent(ctx context.Context, duration time.Duration, found bool) {
	histogram, _ := m.meter.Float64Histogram(
		"app.events.get.duration",
		metric.WithDescription("Get event query duration"),
		metric.WithUnit("s"),
	)
	histogram.Record(ctx, duration.Seconds())

	counter, _ := m.meter.Int64Counter(
		"app.events.get.total",
		metric.WithDescription("Total get event requests"),
		metric.WithUnit("{request}"),
	)
	counter.Add(ctx, 1)

	statusCounter, _ := m.meter.Int64Counter(
		"app.events.get.status",
		metric.WithDescription("Get event status"),
		metric.WithUnit("{status}"),
	)
	if found {
		statusCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "found")))
	} else {
		statusCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "not_found")))
	}
}

func (m *BusinessMetrics) RecordCreateEvent(ctx context.Context, duration time.Duration) {
	histogram, _ := m.meter.Float64Histogram(
		"app.events.create.duration",
		metric.WithDescription("Create event duration"),
		metric.WithUnit("s"),
	)
	histogram.Record(ctx, duration.Seconds())

	counter, _ := m.meter.Int64Counter(
		"app.events.create.total",
		metric.WithDescription("Total create event requests"),
		metric.WithUnit("{request}"),
	)
	counter.Add(ctx, 1)
}

func (m *BusinessMetrics) RecordUpdateEvent(ctx context.Context, duration time.Duration, success bool) {
	histogram, _ := m.meter.Float64Histogram(
		"app.events.update.duration",
		metric.WithDescription("Update event duration"),
		metric.WithUnit("s"),
	)
	histogram.Record(ctx, duration.Seconds())

	counter, _ := m.meter.Int64Counter(
		"app.events.update.total",
		metric.WithDescription("Total update event requests"),
		metric.WithUnit("{request}"),
	)
	counter.Add(ctx, 1)

	statusCounter, _ := m.meter.Int64Counter(
		"app.events.update.status",
		metric.WithDescription("Update event status"),
		metric.WithUnit("{status}"),
	)
	if success {
		statusCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	} else {
		statusCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "failure")))
	}
}

func (m *BusinessMetrics) RecordDeleteEvent(ctx context.Context, duration time.Duration, success bool) {
	histogram, _ := m.meter.Float64Histogram(
		"app.events.delete.duration",
		metric.WithDescription("Delete event duration"),
		metric.WithUnit("s"),
	)
	histogram.Record(ctx, duration.Seconds())

	counter, _ := m.meter.Int64Counter(
		"app.events.delete.total",
		metric.WithDescription("Total delete event requests"),
		metric.WithUnit("{request}"),
	)
	counter.Add(ctx, 1)

	statusCounter, _ := m.meter.Int64Counter(
		"app.events.delete.status",
		metric.WithDescription("Delete event status"),
		metric.WithUnit("{status}"),
	)
	if success {
		statusCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	} else {
		statusCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "failure")))
	}
}

func (m *BusinessMetrics) RecordDatabaseQuery(ctx context.Context, queryType string, duration time.Duration) {
	histogram, _ := m.meter.Float64Histogram(
		"app.db.query.duration",
		metric.WithDescription("Database query duration"),
		metric.WithUnit("s"),
	)
	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("query.type", queryType),
	))
}

// StartHTTPServer starts an HTTP server to serve metrics
func StartHTTPServer(port string, reg *prometheus.Registry) error {
	// Create HTTP handler for prometheus metrics using the registry
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Add health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return server.ListenAndServe()
}

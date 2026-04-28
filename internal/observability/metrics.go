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
	errs := []error{err}

	httpRequestDuration, err = meter.Float64Histogram(
		"app.http.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	errs = append(errs, err)

	httpActiveRequests, err = meter.Int64UpDownCounter(
		"app.http.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	errs = append(errs, err)

	appStartedAt, err = meter.Int64Gauge(
		"app.started_at",
		metric.WithDescription("Application start timestamp (Unix)"),
		metric.WithUnit("{timestamp}"),
	)
	errs = append(errs, err)
	if err := firstError(errs...); err != nil {
		return err
	}

	appStartedAt.Record(context.Background(), time.Now().Unix())

	return nil
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func newFloat64Histogram(meter metric.Meter, name, description, unit string) (metric.Float64Histogram, error) {
	return meter.Float64Histogram(name, metric.WithDescription(description), metric.WithUnit(unit))
}

func newInt64Counter(meter metric.Meter, name, description, unit string) (metric.Int64Counter, error) {
	return meter.Int64Counter(name, metric.WithDescription(description), metric.WithUnit(unit))
}

func RecordRequestMetrics(c *fiber.Ctx, duration time.Duration) {
	if httpRequestsTotal == nil || httpRequestDuration == nil {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("http.method", c.Method()),
		attribute.String("http.path", c.Route().Path),
		attribute.String("http.route", c.Route().Path),
		attribute.Int("http.status_code", c.Response().StatusCode()),
	)

	httpRequestsTotal.Add(c.UserContext(), 1, attrs)
	httpRequestDuration.Record(c.UserContext(), duration.Seconds(), attrs)
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
	listDuration    metric.Float64Histogram
	listTotal       metric.Int64Counter
	getDuration     metric.Float64Histogram
	getTotal        metric.Int64Counter
	getStatus       metric.Int64Counter
	createDuration  metric.Float64Histogram
	createTotal     metric.Int64Counter
	updateDuration  metric.Float64Histogram
	updateTotal     metric.Int64Counter
	updateStatus    metric.Int64Counter
	deleteDuration  metric.Float64Histogram
	deleteTotal     metric.Int64Counter
	deleteStatus    metric.Int64Counter
	dbQueryDuration metric.Float64Histogram
}

func NewBusinessMetrics(serviceName string) (*BusinessMetrics, error) {
	meter := otel.Meter(serviceName + "/business")
	m := &BusinessMetrics{}
	errs := make([]error, 0, 14)
	var err error

	m.listDuration, err = newFloat64Histogram(meter, "app.events.list.duration", "List events query duration", "s")
	errs = append(errs, err)
	m.listTotal, err = newInt64Counter(meter, "app.events.list.total", "Total list events requests", "{request}")
	errs = append(errs, err)
	m.getDuration, err = newFloat64Histogram(meter, "app.events.get.duration", "Get event query duration", "s")
	errs = append(errs, err)
	m.getTotal, err = newInt64Counter(meter, "app.events.get.total", "Total get event requests", "{request}")
	errs = append(errs, err)
	m.getStatus, err = newInt64Counter(meter, "app.events.get.status", "Get event status", "{status}")
	errs = append(errs, err)
	m.createDuration, err = newFloat64Histogram(meter, "app.events.create.duration", "Create event duration", "s")
	errs = append(errs, err)
	m.createTotal, err = newInt64Counter(meter, "app.events.create.total", "Total create event requests", "{request}")
	errs = append(errs, err)
	m.updateDuration, err = newFloat64Histogram(meter, "app.events.update.duration", "Update event duration", "s")
	errs = append(errs, err)
	m.updateTotal, err = newInt64Counter(meter, "app.events.update.total", "Total update event requests", "{request}")
	errs = append(errs, err)
	m.updateStatus, err = newInt64Counter(meter, "app.events.update.status", "Update event status", "{status}")
	errs = append(errs, err)
	m.deleteDuration, err = newFloat64Histogram(meter, "app.events.delete.duration", "Delete event duration", "s")
	errs = append(errs, err)
	m.deleteTotal, err = newInt64Counter(meter, "app.events.delete.total", "Total delete event requests", "{request}")
	errs = append(errs, err)
	m.deleteStatus, err = newInt64Counter(meter, "app.events.delete.status", "Delete event status", "{status}")
	errs = append(errs, err)
	m.dbQueryDuration, err = newFloat64Histogram(meter, "app.db.query.duration", "Database query duration", "s")
	errs = append(errs, err)

	if err := firstError(errs...); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *BusinessMetrics) RecordListEvents(ctx context.Context, duration time.Duration, count int) {
	m.listDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attribute.Int("events.count", count)))
	m.listTotal.Add(ctx, 1)
}

func (m *BusinessMetrics) RecordGetEvent(ctx context.Context, duration time.Duration, found bool) {
	m.getDuration.Record(ctx, duration.Seconds())
	m.getTotal.Add(ctx, 1)
	if found {
		m.getStatus.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "found")))
	} else {
		m.getStatus.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "not_found")))
	}
}

func (m *BusinessMetrics) RecordCreateEvent(ctx context.Context, duration time.Duration) {
	m.createDuration.Record(ctx, duration.Seconds())
	m.createTotal.Add(ctx, 1)
}

func (m *BusinessMetrics) RecordUpdateEvent(ctx context.Context, duration time.Duration, success bool) {
	m.updateDuration.Record(ctx, duration.Seconds())
	m.updateTotal.Add(ctx, 1)
	if success {
		m.updateStatus.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	} else {
		m.updateStatus.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "failure")))
	}
}

func (m *BusinessMetrics) RecordDeleteEvent(ctx context.Context, duration time.Duration, success bool) {
	m.deleteDuration.Record(ctx, duration.Seconds())
	m.deleteTotal.Add(ctx, 1)
	if success {
		m.deleteStatus.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	} else {
		m.deleteStatus.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "failure")))
	}
}

func (m *BusinessMetrics) RecordDatabaseQuery(ctx context.Context, queryType string, duration time.Duration) {
	m.dbQueryDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
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
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return server.ListenAndServe()
}

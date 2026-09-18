package telemetry

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

var uuidSeg = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
var refSeg = regexp.MustCompile(`(?i)inc-\d{4}-\d{4}`)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			MetricsHandler().ServeHTTP(w, r)
			return
		}
		start := time.Now()
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx = WithRequestID(ctx, id)
		route := NormalizeRoute(r.URL.Path)
		lowNoise := route == "/health" || route == "/ready"
		if !lowNoise {
			var spanEnd func()
			ctx2, sp := Start(ctx, "sentinel.http.request",
				attribute.String("http.method", r.Method),
				attribute.String("http.route", route),
			)
			ctx = ctx2
			spanEnd = func() { sp.End() }
			defer spanEnd()
		}
		IncrActive(ctx, HTTPActive, 1, "method", r.Method, "route", route)
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r.WithContext(ctx))
		IncrActive(ctx, HTTPActive, -1, "method", r.Method, "route", route)
		status := strconv.Itoa(sw.status)
		Count(ctx, HTTPRequests, "method", r.Method, "route", route, "status", status)
		Observe(ctx, HTTPDuration, time.Since(start).Seconds(), "method", r.Method, "route", route, "status", status)
	})
}

func NormalizeRoute(path string) string {
	if path == "" {
		return "/"
	}
	p := uuidSeg.ReplaceAllString(path, "{id}")
	p = refSeg.ReplaceAllString(p, "{id}")
	if p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	if p == "" {
		return "/"
	}
	return p
}

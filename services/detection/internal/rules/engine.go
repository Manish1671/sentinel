package rules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/detection/internal/config"
	"github.com/sentinel-dev/sentinel/services/detection/internal/events"
)

const (
	HighLatency     = "high_latency"
	HighErrorRate   = "high_error_rate"
	DBSaturation    = "db_connection_saturation"
	ErrorLogBurst   = "error_log_burst"
	DeploymentAssoc = "deployment_associated"
)

type Hit struct {
	DetectorID  string
	Severity    string
	Title       string
	Summary     string
	Fingerprint string
	Subject     string
	Labels      map[string]string
}

func Fingerprint(detectorID string, serviceID uuid.UUID, subject string) string {
	return detectorID + ":" + serviceID.String() + ":" + subject
}

type Deployment struct {
	ID        string
	Version   string
	StartedAt time.Time
}

// Windows holds short-lived detection state. It is not the source of truth for alerts.
type Windows struct {
	mu      sync.Mutex
	errors  map[uuid.UUID][]time.Time
	deploys map[uuid.UUID][]Deployment
}

func NewWindows() *Windows {
	return &Windows{
		errors:  map[uuid.UUID][]time.Time{},
		deploys: map[uuid.UUID][]Deployment{},
	}
}

func (w *Windows) ObserveError(serviceID uuid.UUID, at time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.errors[serviceID] = append(w.errors[serviceID], at.UTC())
}

func (w *Windows) ObserveDeployment(serviceID uuid.UUID, d Deployment) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.deploys[serviceID] = append(w.deploys[serviceID], d)
}

func (w *Windows) ErrorCount(serviceID uuid.UUID, since time.Time) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, t := range w.errors[serviceID] {
		if !t.Before(since) {
			n++
		}
	}
	return n
}

func (w *Windows) RecentDeployment(serviceID uuid.UUID, at time.Time, window time.Duration) (Deployment, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var best Deployment
	found := false
	for _, d := range w.deploys[serviceID] {
		if d.StartedAt.After(at) {
			continue
		}
		if at.Sub(d.StartedAt) <= window {
			if !found || d.StartedAt.After(best.StartedAt) {
				best = d
				found = true
			}
		}
	}
	return best, found
}

func (w *Windows) Cleanup(now time.Time, retain time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	cutoff := now.Add(-retain)
	for id, ts := range w.errors {
		kept := ts[:0]
		for _, t := range ts {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(w.errors, id)
		} else {
			w.errors[id] = kept
		}
	}
	for id, ds := range w.deploys {
		kept := ds[:0]
		for _, d := range ds {
			if d.StartedAt.After(cutoff) {
				kept = append(kept, d)
			}
		}
		if len(kept) == 0 {
			delete(w.deploys, id)
		} else {
			w.deploys[id] = kept
		}
	}
}

type Engine struct {
	cfg config.Rules
	win *Windows
}

func NewEngine(cfg config.Rules, win *Windows) *Engine {
	if win == nil {
		win = NewWindows()
	}
	return &Engine{cfg: cfg, win: win}
}

func (e *Engine) Windows() *Windows { return e.win }

func (e *Engine) Evaluate(env events.Envelope) []Hit {
	e.win.Cleanup(env.OccurredAt, e.retain())
	switch env.EventType {
	case events.TypeMetric:
		return e.evalMetric(env)
	case events.TypeLog:
		return e.evalLog(env)
	case events.TypeTrace:
		return nil
	case events.TypeDeployment:
		e.observeDeployment(env)
		return nil
	default:
		return nil
	}
}

func (e *Engine) retain() time.Duration {
	r := e.cfg.ErrorBurstWindow
	if e.cfg.DeploymentCorrelationWindow > r {
		r = e.cfg.DeploymentCorrelationWindow
	}
	return r + time.Minute
}

func (e *Engine) evalMetric(env events.Envelope) []Hit {
	svc, err := events.ServiceID(env.Payload)
	if err != nil {
		return nil
	}
	name := strings.ToLower(events.StringField(env.Payload, "name"))
	value, ok := events.FloatField(env.Payload, "value")
	if !ok {
		return nil
	}
	var hits []Hit
	if isLatencyMetric(name) && value > e.cfg.LatencyThresholdMS {
		hits = append(hits, e.hit(HighLatency, "high", svc, name, env,
			"High request latency",
			fmt.Sprintf("Latency metric %s=%.2f exceeded threshold %.2f ms.", name, value, e.cfg.LatencyThresholdMS),
			map[string]string{"metric": name, "value": fmt.Sprintf("%.4f", value)},
		))
	}
	if isErrorRateMetric(name) {
		rate := asRatio(value)
		if rate > e.cfg.ErrorRateThreshold {
			hits = append(hits, e.hit(HighErrorRate, "high", svc, name, env,
				"High error rate",
				fmt.Sprintf("Error-rate metric %s=%.4f exceeded threshold %.4f.", name, rate, e.cfg.ErrorRateThreshold),
				map[string]string{"metric": name, "value": fmt.Sprintf("%.6f", rate)},
			))
		}
	}
	if isDBMetric(name) {
		util := asRatio(value)
		if util > e.cfg.DBConnectionThreshold {
			hits = append(hits, e.hit(DBSaturation, "critical", svc, name, env,
				"Database connection saturation",
				fmt.Sprintf("DB connection utilization %s=%.4f exceeded threshold %.4f.", name, util, e.cfg.DBConnectionThreshold),
				map[string]string{"metric": name, "value": fmt.Sprintf("%.6f", util)},
			))
		}
	}
	return e.annotateDeploy(svc, env.OccurredAt, hits)
}

func (e *Engine) evalLog(env events.Envelope) []Hit {
	svc, err := events.ServiceID(env.Payload)
	if err != nil {
		return nil
	}
	sev := strings.ToLower(events.StringField(env.Payload, "severity"))
	if sev != "error" && sev != "fatal" {
		return nil
	}
	e.win.ObserveError(svc, env.OccurredAt)
	count := e.win.ErrorCount(svc, env.OccurredAt.Add(-e.cfg.ErrorBurstWindow))
	if count < e.cfg.ErrorBurstCount {
		return nil
	}
	hit := e.hit(ErrorLogBurst, e.cfg.ErrorBurstSeverity, svc, "error_logs", env,
		"Repeated error logs",
		fmt.Sprintf("Saw %d error/fatal logs within %s (threshold %d).", count, e.cfg.ErrorBurstWindow, e.cfg.ErrorBurstCount),
		map[string]string{"count": fmt.Sprintf("%d", count)},
	)
	return e.annotateDeploy(svc, env.OccurredAt, []Hit{hit})
}

func (e *Engine) observeDeployment(env events.Envelope) {
	svc, err := events.ServiceID(env.Payload)
	if err != nil {
		return
	}
	id := events.StringField(env.Payload, "deployment_id")
	ver := events.StringField(env.Payload, "version")
	started := env.OccurredAt
	if raw := events.StringField(env.Payload, "started_at"); raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			started = t.UTC()
		} else if t, err := time.Parse(time.RFC3339, raw); err == nil {
			started = t.UTC()
		}
	}
	e.win.ObserveDeployment(svc, Deployment{ID: id, Version: ver, StartedAt: started})
}

func (e *Engine) annotateDeploy(svc uuid.UUID, at time.Time, hits []Hit) []Hit {
	d, ok := e.win.RecentDeployment(svc, at, e.cfg.DeploymentCorrelationWindow)
	if !ok {
		return hits
	}
	for i := range hits {
		if hits[i].Labels == nil {
			hits[i].Labels = map[string]string{}
		}
		hits[i].Labels["deployment_associated"] = "true"
		if d.ID != "" {
			hits[i].Labels["deployment_id"] = d.ID
		}
		if d.Version != "" {
			hits[i].Labels["deployment_version"] = d.Version
		}
		hits[i].Summary = hits[i].Summary + " Correlated with a recent deployment."
	}
	return hits
}

func (e *Engine) hit(detector, severity string, svc uuid.UUID, subject string, env events.Envelope, title, summary string, labels map[string]string) Hit {
	if labels == nil {
		labels = map[string]string{}
	}
	slug := events.StringField(env.Payload, "service_slug")
	if slug != "" {
		labels["service_slug"] = slug
	}
	return Hit{
		DetectorID:  detector,
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		Fingerprint: Fingerprint(detector, svc, subject),
		Subject:     subject,
		Labels:      labels,
	}
}

func isLatencyMetric(name string) bool {
	return strings.Contains(name, "latency")
}

func isErrorRateMetric(name string) bool {
	return strings.Contains(name, "error_rate") || strings.Contains(name, "error-rate")
}

func isDBMetric(name string) bool {
	return strings.Contains(name, "db_connection") ||
		strings.Contains(name, "database_connection") ||
		strings.Contains(name, "connection_pool") ||
		strings.Contains(name, "connection_utilization")
}

func asRatio(v float64) float64 {
	if v > 1 {
		return v / 100
	}
	return v
}

func Catalog(cfg config.Rules) []map[string]any {
	return []map[string]any{
		{"id": HighLatency, "severity": "high", "threshold": cfg.LatencyThresholdMS, "unit": "ms", "description": "Request latency metric exceeds LATENCY_THRESHOLD_MS."},
		{"id": HighErrorRate, "severity": "high", "threshold": cfg.ErrorRateThreshold, "description": "Error-rate metric exceeds ERROR_RATE_THRESHOLD (ratio)."},
		{"id": DBSaturation, "severity": "critical", "threshold": cfg.DBConnectionThreshold, "description": "Database connection utilization exceeds DB_CONNECTION_THRESHOLD."},
		{"id": ErrorLogBurst, "severity": cfg.ErrorBurstSeverity, "count": cfg.ErrorBurstCount, "window_seconds": int(cfg.ErrorBurstWindow.Seconds()), "description": "error/fatal logs exceed count in window."},
		{"id": DeploymentAssoc, "window_seconds": int(cfg.DeploymentCorrelationWindow.Seconds()), "description": "Annotates other alerts when a deployment.created is within the window. Does not create its own alert."},
	}
}

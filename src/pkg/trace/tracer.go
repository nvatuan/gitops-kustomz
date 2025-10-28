package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer
var spanRecorder *SpanRecorder
var outputDir string

// SpanRecorder records spans for human-readable reporting
type SpanRecorder struct {
	spans []spanRecord
}

type spanRecord struct {
	Name     string
	Duration time.Duration
	Start    time.Time
	End      time.Time
	ParentID string
	SpanID   string
}

type SpanInfo struct {
	Name       string     `json:"name"`
	DurationMs float64    `json:"durationMs"`
	Start      string     `json:"start"`
	End        string     `json:"end"`
	Children   []SpanInfo `json:"children,omitempty"`
}

type PerformanceReport struct {
	Spans           []SpanInfo `json:"spans"`
	TotalDurationMs float64    `json:"totalDurationMs"`
	Timestamp       string     `json:"timestamp"`
}

// InitTracer initializes OpenTelemetry tracing
func InitTracer(serviceName string, enabled bool, outDir string) (func(), error) {
	if !enabled {
		// Return no-op shutdown
		return func() {}, nil
	}

	spanRecorder = &SpanRecorder{spans: make([]spanRecord, 0)}
	outputDir = outDir

	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider with span processor that records spans
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(&recordingSpanProcessor{recorder: spanRecorder}),
	)

	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("gitops-kustomz")

	// Return shutdown function
	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Silently fail
		_ = tp.Shutdown(ctx)
		// Export report silently
		_ = ExportReport()
	}

	return shutdown, nil
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name)
}

// GetTracer returns the global tracer
func GetTracer() trace.Tracer {
	return tracer
}

// recordingSpanProcessor records spans for human-readable summary
type recordingSpanProcessor struct {
	recorder *SpanRecorder
}

func (p *recordingSpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {}

func (p *recordingSpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	if p.recorder != nil {
		parentID := ""
		if s.Parent().IsValid() {
			parentID = s.Parent().SpanID().String()
		}
		p.recorder.spans = append(p.recorder.spans, spanRecord{
			Name:     s.Name(),
			Duration: s.EndTime().Sub(s.StartTime()),
			Start:    s.StartTime(),
			End:      s.EndTime(),
			SpanID:   s.SpanContext().SpanID().String(),
			ParentID: parentID,
		})
	}
}

func (p *recordingSpanProcessor) Shutdown(ctx context.Context) error   { return nil }
func (p *recordingSpanProcessor) ForceFlush(ctx context.Context) error { return nil }

// PrintSummary prints a human-readable performance summary
func PrintSummary() {
	// Do nothing - we only export to file now
}

// ExportReport exports the performance report to a JSON file
func ExportReport() error {
	if spanRecorder == nil || len(spanRecorder.spans) == 0 || outputDir == "" {
		return nil
	}

	// Build hierarchy
	hierarchy := buildHierarchy(spanRecorder.spans)

	// Calculate total duration in milliseconds
	totalDurationMs := 0.0
	for _, span := range hierarchy {
		totalDurationMs += span.DurationMs
	}

	report := PerformanceReport{
		Spans:           hierarchy,
		TotalDurationMs: totalDurationMs,
		Timestamp:       time.Now().Format(time.RFC3339Nano),
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	reportPath := filepath.Join(outputDir, "performance-report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

// buildHierarchy converts flat span records into a hierarchical structure
func buildHierarchy(records []spanRecord) []SpanInfo {
	// Create a map of spanID to SpanInfo
	spanMap := make(map[string]*SpanInfo)
	var rootSpans []SpanInfo

	// First pass: create all SpanInfo objects
	for _, record := range records {
		spanInfo := SpanInfo{
			Name:       record.Name,
			DurationMs: float64(record.Duration.Microseconds()) / 1000.0,
			Start:      record.Start.Format(time.RFC3339Nano),
			End:        record.End.Format(time.RFC3339Nano),
			Children:   []SpanInfo{},
		}
		spanMap[record.SpanID] = &spanInfo
	}

	// Second pass: build parent-child relationships
	for _, record := range records {
		if record.ParentID == "" {
			// Root span
			rootSpans = append(rootSpans, *spanMap[record.SpanID])
		} else {
			// Child span - add to parent's children
			if parent, exists := spanMap[record.ParentID]; exists {
				parent.Children = append(parent.Children, *spanMap[record.SpanID])
			}
		}
	}

	// Sort root spans by start time
	sort.Slice(rootSpans, func(i, j int) bool {
		return rootSpans[i].Start < rootSpans[j].Start
	})

	return rootSpans
}

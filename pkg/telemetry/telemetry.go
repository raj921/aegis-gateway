package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type Logger struct {
	file *os.File
}

type AuditLog struct {
	Timestamp   string  `json:"timestamp"`
	TraceID     string  `json:"trace_id"`
	AgentID     string  `json:"agent_id"`
	Tool        string  `json:"tool"`
	Action      string  `json:"action"`
	Decision    bool    `json:"decision_allow"`
	Reason      string  `json:"reason"`
	Version     int     `json:"policy_version"`
	ParamsHash  string  `json:"params_hash"`
	LatencyMs   float64 `json:"latency_ms"`
	ParentAgent string  `json:"parent_agent,omitempty"`
}

var (
	tracer trace.Tracer
	logger *Logger
)

func InitTelemetry(serviceName string, logPath string) error {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return fmt.Errorf("failed to create stdout exporter: %w", err)
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	tracer = tp.Tracer(serviceName)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	logger = &Logger{file: logFile}

	return nil
}

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name)
}

func LogDecision(ctx context.Context, agentID, tool, action, reason, paramsHash, parentAgent string, allowed bool, version int, latencyMs float64) {
	traceID := ""
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
	}

	log := AuditLog{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		TraceID:     traceID,
		AgentID:     agentID,
		Tool:        tool,
		Action:      action,
		Decision:    allowed,
		Reason:      reason,
		Version:     version,
		ParamsHash:  paramsHash,
		LatencyMs:   latencyMs,
		ParentAgent: parentAgent,
	}

	data, _ := json.Marshal(log)
	fmt.Println(string(data))

	if logger != nil && logger.file != nil {
		logger.file.Write(data)
		logger.file.WriteString("\n")
	}
}

func AddSpanAttributes(span trace.Span, attrs map[string]interface{}) {
	var spanAttrs []attribute.KeyValue
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			spanAttrs = append(spanAttrs, attribute.String(k, val))
		case int:
			spanAttrs = append(spanAttrs, attribute.Int(k, val))
		case int64:
			spanAttrs = append(spanAttrs, attribute.Int64(k, val))
		case float64:
			spanAttrs = append(spanAttrs, attribute.Float64(k, val))
		case bool:
			spanAttrs = append(spanAttrs, attribute.Bool(k, val))
		}
	}
	span.SetAttributes(spanAttrs...)
}

func Close() {
	if logger != nil && logger.file != nil {
		logger.file.Close()
	}
}

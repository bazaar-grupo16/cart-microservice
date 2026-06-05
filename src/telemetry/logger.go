package telemetry

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewJSONLogger initializes a Zap production-style logger configured to write JSON to stdout.
func NewJSONLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"

	logger, err := config.Build()
	if err != nil {
		// Fallback to a basic logger if configuration fails
		return zap.NewExample()
	}

	// Add global service field if defined in environment
	if service := os.Getenv("OTEL_SERVICE_NAME"); service != "" {
		logger = logger.With(zap.String("service", service))
	}
	return logger
}

// WithTraceContext extracts trace_id and span_id from the OTel context
// and adds them as fields to the Zap logger.
func WithTraceContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return logger
	}
	return logger.With(
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	)
}

// GinAccessLogger creates a Gin middleware that logs HTTP requests using Zap in JSON format.
func GinAccessLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		l := WithTraceContext(c.Request.Context(), logger)
		l.Info("http access",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", latency.Milliseconds()),
			zap.String("client_ip", c.ClientIP()),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)
	}
}

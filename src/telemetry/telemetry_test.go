package telemetry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bazaar-grupo16/cart-microservice/src/service"
	"github.com/bazaar-grupo16/cart-microservice/src/telemetry"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// setupTestTracer configures a local SDK TracerProvider so spans generate valid trace/span IDs in tests.
func setupTestTracer() {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
}

func TestTracerInitializes(t *testing.T) {
	os.Setenv("OTEL_SERVICE_NAME", "test-cart")
	os.Setenv("OTEL_DEPLOYMENT_ENVIRONMENT", "test")
	ctx := context.Background()

	shutdown, err := telemetry.InitTracer(ctx)
	if err != nil {
		t.Fatalf("expected no error initializing tracer, got %v", err)
	}
	defer shutdown(ctx)

	// Verify global TracerProvider is set
	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Error("expected global TracerProvider to be configured, got nil")
	}
}

func TestCorrelationMiddleware_ReadsIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(telemetry.CorrelationMiddleware())

	var capturedCorrelationID string
	r.GET("/test", func(c *gin.Context) {
		capturedCorrelationID = c.GetString("correlation_id")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "custom-correlation-id-123")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if capturedCorrelationID != "custom-correlation-id-123" {
		t.Errorf("expected correlation_id 'custom-correlation-id-123', got '%s'", capturedCorrelationID)
	}
	if w.Header().Get("X-Correlation-ID") != "custom-correlation-id-123" {
		t.Errorf("expected response header X-Correlation-ID 'custom-correlation-id-123', got '%s'", w.Header().Get("X-Correlation-ID"))
	}
}

func TestCorrelationMiddleware_FallsBackToTraceID(t *testing.T) {
	setupTestTracer()
	gin.SetMode(gin.TestMode)

	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("test")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Start a dummy span to simulate otelgin middleware running first
		ctx, span := tracer.Start(c.Request.Context(), "root")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.Use(telemetry.CorrelationMiddleware())

	var capturedCorrelationID string
	var traceID string
	r.GET("/test", func(c *gin.Context) {
		capturedCorrelationID = c.GetString("correlation_id")
		span := trace.SpanFromContext(c.Request.Context())
		traceID = span.SpanContext().TraceID().String()
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if capturedCorrelationID == "" {
		t.Error("expected correlation_id to fall back to trace_id, got empty")
	}
	if capturedCorrelationID != traceID {
		t.Errorf("expected correlation_id to match trace_id '%s', got '%s'", traceID, capturedCorrelationID)
	}
}

func TestBuildObservabilityHeaders_IncludesTraceparent(t *testing.T) {
	setupTestTracer()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("test")

	var headers map[string]string
	r.GET("/test", func(c *gin.Context) {
		// Simulate correlation ID setting
		c.Set("correlation_id", "my-correlation-id")

		ctx, span := tracer.Start(c.Request.Context(), "subspan")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		headers = telemetry.BuildObservabilityHeaders(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if headers["X-Correlation-ID"] != "my-correlation-id" {
		t.Errorf("expected X-Correlation-ID header 'my-correlation-id', got '%s'", headers["X-Correlation-ID"])
	}
	if headers["traceparent"] == "" {
		t.Error("expected traceparent header to be populated, got empty")
	}
}

func TestMetricsEndpoint_Returns200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", telemetry.MetricsHandler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") == "" {
		t.Error("expected Content-Type header to be present")
	}
}

func TestPropagatesCorrelationID_ToCatalog(t *testing.T) {
	setupTestTracer()

	// 1. Start a mock server for Catalog
	var receivedHeaders http.Header
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"id":"prod1","title":"Product 1","price":10.5,"stock":10,"status":"active"}}`))
	}))
	defer mockServer.Close()

	// 2. Initialize CatalogClient pointing to mock server
	client := service.NewCatalogClient(mockServer.URL)

	// 3. Create context with trace and correlation ID
	ctx := context.Background()
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	// Create dummy gin context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request = c.Request.WithContext(ctx)
	c.Set("correlation_id", "test-corr-id-789")

	// Call GetProduct passing the Context (which contains our telemetry headers)
	_, err := client.GetProduct(c, "prod1")
	if err != nil {
		t.Fatalf("expected no error calling GetProduct, got %v", err)
	}

	if receivedHeaders == nil {
		t.Fatal("expected mock server to receive headers, but it did not")
	}

	if receivedHeaders.Get("X-Correlation-ID") != "test-corr-id-789" {
		t.Errorf("expected outgoing X-Correlation-ID 'test-corr-id-789', got '%s'", receivedHeaders.Get("X-Correlation-ID"))
	}
	if receivedHeaders.Get("traceparent") == "" {
		t.Error("expected traceparent header to be propagated, got empty")
	}
}

// serviceB/main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    // "go.opentelemetry.io/otel/trace"
    "github.com/gorilla/mux"
)

const TraceIDHeader = "Trace-Id"

func initTracer() func() {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
    if err != nil {
        log.Fatal(err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("service-b"),
        )),
    )
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.TraceContext{})

    return func() {
        _ = tp.Shutdown(context.Background())
    }
}

func main() {
    shutdown := initTracer()
    defer shutdown()

    r := mux.NewRouter()
    r.HandleFunc("/process", func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()
        tracer := otel.Tracer("service-b")

        // 从 header 提取 trace
        ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

        ctx, span := tracer.Start(ctx, "service-b-span")
        defer span.End()

        // 获取 trace id 并返回给调用方 (header + body)
        traceID := span.SpanContext().TraceID().String()
        if traceID != "" {
            w.Header().Set(TraceIDHeader, traceID)
        }

        time.Sleep(1000 * time.Millisecond) // 模拟处理时间

        // 保留简洁的文本响应，同时包含 trace id 以便客户端能看到
        fmt.Fprintf(w, "Hello from Service B (trace_id=%s)\n", traceID)
    })

    http.ListenAndServe(":8081", r)
}

// serviceA/main.go
package main

import (
    "context"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
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
            semconv.ServiceNameKey.String("service-a"),
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
    r.HandleFunc("/callb", func(w http.ResponseWriter, req *http.Request) {
        ctx := req.Context()
        tracer := otel.Tracer("service-a")

        // Check if Trace-Id is present in the request headers
        traceID := req.Header.Get(TraceIDHeader)
        if traceID == "" {
            // Generate a new Trace-ID if not present
            _, span := tracer.Start(ctx, "service-a-span")
            defer span.End()
            traceID = span.SpanContext().TraceID().String()
        } else {
            // Use the existing Trace-ID
            ctx = propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(req.Header))
        }

        if traceID != "" {
            w.Header().Set(TraceIDHeader, traceID)
        }

        // 调用服务 B
        client := &http.Client{}
        reqB, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8081/process", nil)
        // 传递 trace 信息
        otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(reqB.Header))
        resp, err := client.Do(reqB)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        body, _ := ioutil.ReadAll(resp.Body)

        // 响应内容包含来自 B 的原始 body 和统一的 trace id 信息（只显示一个 trace id）
        fmt.Fprintf(w, "Service A received: %s\n Trace-Id = %s \n", string(body), traceID)

        time.Sleep(2000 * time.Millisecond) // 模拟处理时间
    })

    http.ListenAndServe(":8080", r)
}

// serviceA/main.go
package main

import (
    "context"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    // "os"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    // "go.opentelemetry.io/otel/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "github.com/gorilla/mux"
)

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

        // 使用 TraceID
        ctx, span := tracer.Start(ctx, "service-a-span")
        defer span.End()

        // 从请求 header 获取 traceparent (记录)
        traceparent := req.Header.Get("traceparent")
        if traceparent != "" {
            log.Println("Received traceparent:", traceparent)
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

        // 从当前 span 获取 trace id
        traceID := span.SpanContext().TraceID().String()
        // 如果 B 返回了 trace header，也读取出来以便在响应中展示
        bTrace := resp.Header.Get("Trace-Id")

        // 将 trace id 返回给最终客户端（header + body）
        if traceID != "" {
            w.Header().Set("Trace-Id", traceID)
        }
        if bTrace != "" {
            w.Header().Set("Trace-Id-From-B", bTrace)
        }

        // 响应内容包含来自 B 的原始 body 和 trace id 信息
        fmt.Fprintf(w, "Service A received: %s\nservice-a-trace_id=%s\nservice-b-trace_id=%s", string(body), traceID, bTrace)

        time.Sleep(2000 * time.Millisecond) // 模拟处理时间
    })

    http.ListenAndServe(":8080", r)
}

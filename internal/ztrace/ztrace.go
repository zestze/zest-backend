package ztrace

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer trace.Tracer

type Options struct {
	ServiceName   string
	OTLPEndppoint string
}

func New(ctx context.Context, opts Options) (*sdktrace.TracerProvider, error) {
	//exporter, err := stdout.New(stdout.WithPrettyPrint())
	exporter, err := newOTLPExporter(ctx, opts.OTLPEndppoint)
	if err != nil {
		return nil, err
	}

	tp, err := newTraceProvider(exporter, opts)
	if err != nil {
		return nil, err
	}
	// set `otel` global
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{},
		propagation.Baggage{}))
	// set my global
	tracer = tp.Tracer(opts.ServiceName)
	return tp, err
}

func Start(ctx context.Context, name string) (context.Context, trace.Span) {
	// gin middleware oddity (this is just a hack for now)
	// the span is stored in the request's context exclusively
	if c, ok := ctx.(*gin.Context); ok {
		return tracer.Start(c.Request.Context(), name)
	}
	return tracer.Start(ctx, name)
}

func newTraceProvider(exporter sdktrace.SpanExporter, opts Options) (*sdktrace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(opts.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(r),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	), nil
}

func newOTLPExporter(ctx context.Context, otlpEndpoint string) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx, otlptracehttp.WithInsecure(), otlptracehttp.WithEndpoint(otlpEndpoint))
	/*
		driver := otlpgrpc.NewDriver(
			otlpgrpc.WithInsecure(), // insecure is fine, localhost operations
			otlpgrpc.WithEndpoint(otlpEndpoint),
			otlpgrpc.WithDialOption(otlpgrpc.WithBlock()), // TODO(Zeke): not super sure what this does!
		)
		return otlp.NewExporter(ctx, driver)
	*/
}

// Package otel initialises OpenTelemetry TracerProvider and MeterProvider from
// application configuration, returning a single shutdown closure that flushes
// both signals to OTLP/gRPC.
package otel

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	otelprom "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otelcontribruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"google.golang.org/grpc"
)

// ipv4Dialer returns a grpc.DialOption that forces IPv4 dialing on the
// OTLP/gRPC connection.
//
// Cloud Run's network egress — both Serverless VPC Access and Direct
// VPC Egress — historically terminates the IPv6 path. When the OTLP
// dialer uses Go's default DNS resolver (Happy Eyeballs, RFC 6555),
// the resolver prefers the AAAA record returned by the endpoint's
// authoritative DNS; the dial then hangs until the per-attempt deadline
// expires before the IPv4 fallback completes. The result is
// `dial tcp [2001:4860:4844:400::]:4317: i/o timeout` failures on the
// metric export, even though the destination is reachable over IPv4.
//
// Forcing the dialer to tcp4 short-circuits the IPv6 attempt entirely
// and connects over the IPv4 route that Cloud Run's VPC egress
// supports. This applies regardless of whether the endpoint is the
// sidecar collector (Cloud Run sidecar pattern, on `localhost:4317`)
// or a private collector reachable via Serverless VPC Access / Direct
// VPC Egress — both rely on the same IPv4-only egress path.
//
// The dialer is unconditionally applied. On developer machines and
// local collectors it has no effect (IPv4 is already the default);
// on Cloud Run it sidesteps the IPv6 timeout.
func ipv4Dialer() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp4", addr)
	})
}// noopShutdown is returned when OTel is disabled or when the providers fail
// to construct. Both call sites are safe: the returned function can be
// invoked any number of times without panicking.
func noopShutdown(_ context.Context) error { return nil }

// resourceAttrs is the bag of inputs used to construct a Resource. Kept as
// a single value so the build helper stays pure and easy to unit-test.
type resourceAttrs struct {
	serviceName        string
	resourceAttributes string
	revision           string
	service            string
	version            string
}

// Init constructs OpenTelemetry TracerProvider and MeterProvider using
// configuration drawn from cfg, exporting via OTLP/gRPC. Both providers
// are set as the OTel globals so the contrib instrumentations (otelhttp,
// otelgrpc) and any explicit otel.Meter("...") calls pick them up.
//
// On success, the returned shutdown closure flushes both providers and is
// safe to call multiple times. The caller is responsible for invoking it
// before process exit (typically via defer in cmd/serve.go).
//
// When cfg.OTELEnabled is false, Init returns a no-op shutdown and does
// not touch the OTel globals.
func Init(ctx context.Context) (func(context.Context) error, error) {
	if !cfg.OTELEnabled.Value() {
		return noopShutdown, nil
	}

	// Surface Cloud Run identity in the resource when present.
	res, err := buildResource(ctx, resourceAttrs{
		serviceName:        cfg.OTELServiceName.Value(),
		resourceAttributes: cfg.OTELResourceAttributes.Value(),
		revision:           os.Getenv("K_REVISION"),
		service:            os.Getenv("K_SERVICE"),
		version:            version.Version,
	})
	if err != nil {
		return noopShutdown, fmt.Errorf("build resource: %w", err)
	}

	tp, err := buildTracerProvider(ctx, res)
	if err != nil {
		return noopShutdown, fmt.Errorf("build tracer provider: %w", err)
	}

	mp, err := buildMeterProvider(ctx, res)
	if err != nil {
		// Best-effort shutdown of the tracer provider before returning.
		_ = tp.Shutdown(ctx)
		return noopShutdown, fmt.Errorf("build meter provider: %w", err)
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Runtime metrics are best-effort. A failure here is logged but does
	// not roll back the providers.
	if err := otelcontribruntime.Start(otelcontribruntime.WithMeterProvider(mp)); err != nil {
		log.Printf("otel: failed to start runtime instrumentation: %v", err)
	}

	return func(ctx context.Context) error {
		var errs []string
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("tracer: %v", err))
		}
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("meter: %v", err))
		}
		// The contrib runtime instrumentation spawns a goroutine that runs
		// for the lifetime of the process; there is no Stop function
		// exposed. It dies with the process.
		if len(errs) > 0 {
			return fmt.Errorf("otel shutdown: %s", strings.Join(errs, "; "))
		}
		return nil
	}, nil
}

// buildTracerProvider builds a TracerProvider backed by an OTLP/gRPC batch
// span processor. Sampling is AlwaysOn per stakeholder decision; this can
// be moved behind a cfg flag in a follow-up.
func buildTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTELExporterEndpoint.Value()),
		otlptracegrpc.WithDialOption(ipv4Dialer()),
	}
	// cfg.OTELExporterInsecure defaults to true for sidecar / local
	// collectors. Set to false (and supply a TLS-fronted endpoint such
	// as cloudtrace.googleapis.com:443) when exporting directly to
	// Google's public API.
	if cfg.OTELExporterInsecure.Value() {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	), nil
}

// buildMeterProvider builds a MeterProvider backed by an OTLP/gRPC periodic
// reader. The default export interval is the SDK default (60s).
func buildMeterProvider(ctx context.Context, res *resource.Resource) (*otelprom.MeterProvider, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTELExporterEndpoint.Value()),
		otlpmetricgrpc.WithDialOption(ipv4Dialer()),
	}
	if cfg.OTELExporterInsecure.Value() {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	exp, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return otelprom.NewMeterProvider(
		otelprom.WithReader(otelprom.NewPeriodicReader(exp)),
		otelprom.WithResource(res),
	), nil
}

// buildResource composes the OpenTelemetry Resource used by both providers.
//
// The function is exported only via the test in init_test.go. Production
// callers should use Init; this helper exists so the resource
// construction logic can be unit-tested without constructing live
// providers.
func buildResource(ctx context.Context, in resourceAttrs) (*resource.Resource, error) {
	// Start with what the OTel SDK auto-detects: env vars (OTEL_RESOURCE_ATTRIBUTES,
	// OTEL_SERVICE_NAME, etc.), host info, process info, telemetry SDK info.
	auto, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceName(in.serviceName),
	}

	if in.version != "" {
		attrs = append(attrs, semconv.ServiceVersion(in.version))
	}
	if in.revision != "" {
		attrs = append(attrs, attribute.String("cloud.run.revision", in.revision))
	}
	if in.service != "" {
		attrs = append(attrs, attribute.String("cloud.run.service", in.service))
	}
	attrs = append(attrs, parseResourceAttributes(in.resourceAttributes)...)

	merged, err := resource.Merge(auto, resource.NewSchemaless(attrs...))
	if err != nil {
		return nil, err
	}

	return merged, nil
}

// parseResourceAttributes parses a comma-separated `key=value` string into
// OTel attribute key-value pairs. Malformed entries are silently dropped.
// Empty values are emitted (an explicit "key=" produces an attribute with
// the empty string value), matching the OTel SDK's
// `OTEL_RESOURCE_ATTRIBUTES` parsing behaviour.
func parseResourceAttributes(in string) []attribute.KeyValue {
	if in == "" {
		return nil
	}

	var out []attribute.KeyValue
	for _, raw := range strings.Split(in, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		eq := strings.IndexByte(raw, '=')
		if eq <= 0 || eq == len(raw)-1 {
			// No `=`, or `=` is the first or last character. Skip.
			continue
		}

		k := strings.TrimSpace(raw[:eq])
		v := strings.TrimSpace(raw[eq+1:])
		if k == "" {
			continue
		}

		out = append(out, attribute.String(k, v))
	}

	return out
}

// Package otel initialises OpenTelemetry TracerProvider and MeterProvider from
// application configuration, returning a single shutdown closure that flushes
// both signals to OTLP/gRPC.
package otel

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/version"
	"go.opentelemetry.io/contrib/detectors/gcp"
	otelcontribruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	otelprom "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
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
}

// otlpProbeTimeout bounds the startup probe to avoid slowing down
// server startup when the OTLP endpoint is unreachable in a remote
// network (e.g., behind a slow firewall). 3 seconds is long enough
// to cover a TLS handshake over a high-latency link but short enough
// to fail fast on the common "no listener" case.
const otlpProbeTimeout = 3 * time.Second

// probeOTLPEndpoint verifies that the configured OTLP endpoint accepts
// a transport connection. It performs a TLS handshake for secure
// endpoints, but does not send an OTLP request or validate credentials,
// IAM permissions, resource attributes, or API compatibility.
//
// The probe uses the same tcp4 network family as the production dialer,
// so it catches:
//   - "no listener" (connection refused)
//   - DNS resolution failure
//   - IPv6 timeout (vs the IPv4 route Cloud Run supports)
//   - TLS handshake and certificate failures for secure endpoints
func probeOTLPEndpoint(ctx context.Context, endpoint otlpEndpoint) error {
	probeCtx, cancel := context.WithTimeout(ctx, otlpProbeTimeout)
	defer cancel()

	var (
		conn net.Conn
		err  error
	)
	if endpoint.insecure {
		conn, err = (&net.Dialer{}).DialContext(probeCtx, "tcp4", endpoint.target)
	} else {
		dialer := &tls.Dialer{
			NetDialer: &net.Dialer{},
			Config: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: endpoint.hostname,
			},
		}
		conn, err = dialer.DialContext(probeCtx, "tcp4", endpoint.target)
	}
	if err != nil {
		return err
	}
	return conn.Close()
}

type otlpEndpoint struct {
	target   string
	hostname string
	insecure bool
	fromURL  bool
}

func resolveOTLPEndpoint(raw string, insecure bool) (otlpEndpoint, error) {
	endpoint := otlpEndpoint{target: raw, insecure: insecure}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" {
			return otlpEndpoint{}, fmt.Errorf("invalid OTLP endpoint %q", raw)
		}
		if u.Path != "" && u.Path != "/" {
			return otlpEndpoint{}, fmt.Errorf("OTLP/gRPC endpoint %q must not contain a path", raw)
		}
		switch u.Scheme {
		case "http":
			endpoint.insecure = true
		case "https":
			endpoint.insecure = false
		default:
			return otlpEndpoint{}, fmt.Errorf("unsupported OTLP endpoint scheme %q", u.Scheme)
		}

		port := u.Port()
		if port == "" {
			if endpoint.insecure {
				port = "80"
			} else {
				port = "443"
			}
		}
		endpoint.target = net.JoinHostPort(u.Hostname(), port)
		endpoint.hostname = u.Hostname()
		endpoint.fromURL = true
		return endpoint, nil
	}

	host, _, err := net.SplitHostPort(raw)
	if err != nil {
		return otlpEndpoint{}, fmt.Errorf("OTLP endpoint %q must include a port: %w", raw, err)
	}
	endpoint.hostname = host
	return endpoint, nil
}

// Google Telemetry API scopes for OAuth token acquisition.
const (
	// Trace ingestion. Maps to IAM role roles/telemetry.tracesWriter.
	traceOTLPScope = "https://www.googleapis.com/auth/trace.append"
	// Metrics ingestion. Maps to IAM role roles/monitoring.metricWriter.
	meterOTLPScope = "https://www.googleapis.com/auth/monitoring.write"
)

// otlpCredentials holds an optional per-RPC OAuth credentials for the
// OTLP gRPC exporter. When `enabled` is false, no credentials are
// attached and the exporter dials without per-RPC auth (correct for
// sidecar / local collectors on a trusted loopback). When true, the
// credentials are attached to every gRPC call; the first invocation
// triggers a token fetch via Application Default Credentials.
//
// Construction is intentionally lazy: oauth.NewApplicationDefault does
// not block on a network call at construction time, only when the
// first token is fetched. So even production code paths that end up
// skipping the first export don't pay any startup cost.
type otlpCredentials struct {
	creds   credentials.PerRPCCredentials
	enabled bool
}

// newOTLPCredentials constructs per-RPC OAuth credentials for the OTLP
// exporter when (and only when) the endpoint is the Google Telemetry
// API. For sidecar / local / remote-collector endpoints, no
// credentials are attached.
func newOTLPCredentials(ctx context.Context, hostname, scope string) (otlpCredentials, error) {
	if hostname != "googleapis.com" && !strings.HasSuffix(hostname, ".googleapis.com") {
		return otlpCredentials{enabled: false}, nil
	}
	c, err := oauth.NewApplicationDefault(ctx, scope)
	if err != nil {
		return otlpCredentials{}, err
	}
	return otlpCredentials{creds: c, enabled: true}, nil
}

// noopShutdown is returned when OTel is disabled or when the providers fail
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

	endpoint, err := resolveOTLPEndpoint(
		cfg.OTELExporterEndpoint.Value(),
		cfg.OTELExporterInsecure.Value(),
	)
	if err != nil {
		return noopShutdown, err
	}

	// Per-RPC OAuth credentials for the Google Telemetry API. These
	// are only attached when the endpoint is a Google one (i.e. it
	// contains "googleapis.com"); for sidecar / local collectors, the
	// token source is skipped and the exporter dials unauthenticated.
	// Construction is lazy; no token fetch happens at this point.
	traceCreds, err := newOTLPCredentials(ctx, endpoint.hostname, traceOTLPScope)
	if err != nil {
		return noopShutdown, fmt.Errorf("build trace credentials: %w", err)
	}
	meterCreds, err := newOTLPCredentials(ctx, endpoint.hostname, meterOTLPScope)
	if err != nil {
		return noopShutdown, fmt.Errorf("build meter credentials: %w", err)
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

	tp, err := buildTracerProvider(ctx, res, endpoint, traceCreds)
	if err != nil {
		return noopShutdown, fmt.Errorf("build tracer provider: %w", err)
	}

	mp, err := buildMeterProvider(ctx, res, endpoint, meterCreds)
	if err != nil {
		// Best-effort shutdown of the tracer provider before returning.
		_ = tp.Shutdown(ctx)
		return noopShutdown, fmt.Errorf("build meter provider: %w", err)
	}

	// Startup probe: fail fast when the configured OTLP endpoint is
	// unreachable. Cloud Run's OTel pipeline silently drops all
	// spans and metrics if the collector isn't listening (or DNS is
	// wrong, or the IPv6 path hangs) — the application runs fine but
	// produces no observability data, and the failure only surfaces
	// in Cloud Logging hours later. Doing a transport probe at start
	// surfaces the misconfiguration immediately and prevents the
	// server from looking healthy when it isn't.
	//
	// The probe is intentionally transport-only: it catches the common
	// failure modes (no listener, DNS failure, IPv6 timeout, and TLS
	// handshake failure). Endpoint
	// configuration errors that produce Unimplemented responses
	// (e.g. wrong service path) surface in the per-export log
	// messages, which are loud enough to be diagnosed quickly.
	if cfg.OTELProbeEndpoint.Value() {
		if err := probeOTLPEndpoint(ctx, endpoint); err != nil {
			_ = tp.Shutdown(ctx)
			_ = mp.Shutdown(ctx)
			return noopShutdown, fmt.Errorf(
				"startup probe to OTLP endpoint %q failed: %w\n"+
					"hint: confirm the endpoint is reachable and its TLS certificate is valid;\n"+
					"      this probe does not validate OTLP ingestion, credentials, or IAM;\n"+
					"      bypass it with --otel.probe-endpoint=false if intentionally disabled",
				endpoint.target, err,
			)
		}
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
//
// creds carries per-RPC OAuth credentials. When creds.enabled is true
// (i.e. when the endpoint is the Google Telemetry API), the token
// source is attached to every gRPC call.
func buildTracerProvider(ctx context.Context, res *resource.Resource, endpoint otlpEndpoint, creds otlpCredentials) (*sdktrace.TracerProvider, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint.target),
		otlptracegrpc.WithDialOption(ipv4Dialer()),
	}
	if creds.enabled {
		opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithPerRPCCredentials(creds.creds)))
	}
	if endpoint.insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(nil)))
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
func buildMeterProvider(ctx context.Context, res *resource.Resource, endpoint otlpEndpoint, creds otlpCredentials) (*otelprom.MeterProvider, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint.target),
		otlpmetricgrpc.WithDialOption(ipv4Dialer()),
	}
	if creds.enabled {
		opts = append(opts, otlpmetricgrpc.WithDialOption(grpc.WithPerRPCCredentials(creds.creds)))
	}
	if endpoint.insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	} else {
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(nil)))
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
	// The GCP detector is only registered when we're actually on Cloud Run
	// (signalled by the K_SERVICE env var, which the Cloud Run runtime
	// sets on every instance). Otherwise the detector's metadata-server
	// ping is a slow network call against a non-existent endpoint, which
	// would noticeably slow startup on developer machines.
	opts := []resource.Option{
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
	}
	if os.Getenv("K_SERVICE") != "" {
		opts = append(opts, resource.WithDetectors(gcp.NewDetector()))
	}
	auto, err := resource.New(ctx, opts...)
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
	if projectID := gcpProjectID(auto); projectID != "" {
		attrs = append(attrs, attribute.String("gcp.project_id", projectID))
	}
	attrs = append(attrs, parseResourceAttributes(in.resourceAttributes)...)

	merged, err := resource.Merge(auto, resource.NewSchemaless(attrs...))
	if err != nil {
		return nil, err
	}

	return merged, nil
}

func gcpProjectID(res *resource.Resource) string {
	value, ok := res.Set().Value(semconv.CloudAccountIDKey)
	if !ok {
		return ""
	}
	return value.AsString()
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

# Observe Locally with OpenTelemetry

The server emits OpenTelemetry traces and metrics for every
inbound request, every storage operation, and the Go runtime.
The data is exported via OTLP/gRPC to the endpoint configured by
the standard `OTEL_EXPORTER_OTLP_ENDPOINT` env var (or the
`--otel.exporter.endpoint` flag, which defaults to
`cloudtrace.googleapis.com:4317`).

This guide shows how to route the data to a local collector for
development.

## Prerequisites

* The `otelcol` binary from the [OpenTelemetry Collector
  Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib)
  distribution installed and on your `$PATH`. The binary is also
  installable via `brew install otelcol` on macOS or by downloading
  a release artifact from
  [the upstream releases page](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases).

## Run a local collector

The minimum configuration below accepts OTLP/gRPC on
`localhost:4317` and writes the data to a file on disk so the
tests can be replayed or inspected.

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: localhost:4317

exporters:
  file:
    path: /tmp/x40-link-otlp.jsonl
  debug:

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [file, debug]
    metrics:
      receivers: [otlp]
      exporters: [file, debug]
```

Start the collector with:

```bash
otelcol --config otel-collector-config.yaml
```

## Run the server pointed at the local collector

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
    ./x40.link serve --storage.boltdb.file /tmp/x40.link.db
```

Spans and metrics should now appear in `/tmp/x40-link-otlp.jsonl`
and on the collector's debug output.

## What the default endpoint is for

The default `--otel.exporter.endpoint` is `localhost:4317`. This
matches the [Cloud Run sidecar pattern](https://cloud.google.com/run/docs/deploying#sidecars),
where the OpenTelemetry Collector runs as a sidecar container on
the same instance as `x40.link` and listens on the loopback
interface. The application talks to it in plaintext; the sidecar
forwards to Google Cloud Trace and Cloud Monitoring with TLS over
Google's internal network.

If you want to export directly to Cloud Trace instead of via a
sidecar, set `--otel.exporter.endpoint=cloudtrace.googleapis.com:443`
and `--otel.exporter.insecure=false`. The endpoint must be `:443`
(Google's public API runs TLS on the standard HTTPS port) and
`insecure=false` so the OTLP exporter performs the TLS handshake.

## What you should see

* **Span names** are business-operation names — `create_link`,
  `resolve_link`, `redirect`, `storage.lookup`, `storage.write` —
  not the underlying gRPC method name or HTTP path. The gRPC and
  HTTP instrumentation libraries are configured to rename
  otelhttp/otelgrpc's default span names to match AGENTS.md.
* **Metric names** are `x40.link.links.created`,
  `x40.link.links.resolved`, `x40.link.links.not_found`, and
  `x40.link.storage.errors`. Each carries a `storage` label
  (e.g. `boltdb`, `hashmap`, `yaml`, `firestore`) and the
  resolved/not_found metrics also carry a `surface` label
  (`http` or `grpc`).
* **Go runtime metrics** (`process.runtime.go.goroutines`,
  `process.runtime.go.mem.heap_*`, `process.runtime.go.gc.*`) are
  emitted under the `x40.link/otel` instrument name.
* **HTTP semantic-convention metrics** (`http.server.request.duration`,
  `http.server.requests`) and **gRPC** (`rpc.server.duration`,
  `rpc.server.requests`) are added automatically by the contrib
  libraries (`otelhttp`, `otelgrpc`).

## Disabling OTel

For pure local debugging, you can turn the OTel pipeline off
entirely:

```bash
./x40.link serve --otel.enabled=false --storage.boltdb.file /tmp/x40.link.db
```

The server will continue to serve traffic; you simply lose the
observability surface. This is useful when you're trying to
isolate behaviour and don't want traces/metrics polluting a
debugger session.

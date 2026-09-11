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

The default `--otel.exporter.endpoint` is `telemetry.googleapis.com:443`
with TLS — i.e., out of the box the application exports directly to
the unified **Google Telemetry API** over OTLP/gRPC. Authentication
is handled via Application Default Credentials — in Cloud Run, the
service account's metadata server; in dev with the SDK on a
machine with no Google connection, you'd flip the flags to point
at a local collector (see below).

**Note on legacy endpoints:** `cloudtrace.googleapis.com` does not
accept OTLP/gRPC — it speaks the older `google.devtools.cloudtrace.v2`
protobuf. The default endpoint in this codebase is therefore the
Telemetry API, not the legacy Cloud Trace endpoint.

If you're running locally without a Cloud Run metadata server, you
have two options:

* **Point at a local OTel collector** (the easiest dev path). Run
  `otelcol` locally on `localhost:4317` with the collector config
  below, then start the server with:
  ```bash
  ./x40.link serve \
      --otel.exporter.endpoint=localhost:4317 \
      --otel.exporter.insecure=true \
      --storage.boltdb.file /tmp/x40.link.db
  ```
  The `--otel.exporter.insecure=true` flag is required because
  localhost loopback traffic doesn't need TLS, and the OTel
  collector typically listens in plaintext on the loopback.

* **Bypass the exporter entirely.** Use `--otel.enabled=false` to
  skip all observability work while debugging.

The sidecar collector pattern (Cloud Run sidecar listening on
`localhost:4317`) uses the same flags as the local-collector dev
path: endpoint `localhost:4317`, insecure `true`.

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

# Development

This document is for engineers working on the x40.link codebase. It covers
project layout, design notes, and behavior that isn't obvious from reading
the code.

## CLI Subcommands

The CLI binary lives in `cli/`. It exposes two subcommands:

* **`@ <url>`** (root command) — create a short link. Requires OAuth
  credentials via the device authorization flow. See `cli/main.go::DoURL`.
* **`@ resolve <url>`** — look up the destination of a short link. Does
  *not* require OAuth credentials. See `cli/main.go::DoResolve` and
  `cli/main.go::doResolveWithClient`.

The flag sets are split into `apiFlagSet` (just `cfg.APIEndpoint`) and
`authFlagSet` (the OAuth-related flags). The root command uses both
(composed into `urlFlagSet`); the `resolve` subcommand uses only the
`apiFlagSet`. Adding a new subcommand that needs a different set of
configuration is a matter of attaching the right flag set to the new
cobra command.

## Public vs. Authenticated gRPC Methods

The gRPC server in `api/dev/` enforces OAuth scope on a per-method basis,
via the JWT server interceptor in `api/auth/jwts/`. A method's scope is
declared in the proto file with the `oauth2_scope` extension. The
interceptor reads those scopes via `api/api.go::X40Permissions()` and
enforces them at call time.

A method whose declared scope is *absent* (or, equivalently, declared as
the empty string) is treated as **publicly callable**:

* No `Authorization` header is required.
* If one is supplied, it is still stripped from the outgoing context so
  the handler does not see credentials it doesn't need.
* No `storage.CtxKeyAgent` is attached to the context. Handlers of
  public methods must not assume an authenticated agent.

The `Get` method on `x40.dev.url.ManageURLs` is currently the only public
method. The destination of a short link is functionally public information
— the HTTP redirect at `server/storage.go::Redirect` already discloses
it to anonymous users — so the gRPC `Get` RPC aligns with that reality by
being public. This is what allows the `resolve` subcommand to work
without OAuth.

When adding a new RPC, ask: is the response of this RPC already disclosed
to anonymous users by another path (e.g., the HTTP redirect handler, a
public website, etc.)? If so, declaring it as public — by omitting the
`oauth2_scope` extension on the method — is the right call. If the
response carries data that is genuinely not public, declare a scope.

There is a TODO in `api/dev/url.proto` acknowledging that "authentication
should be an emergent property of these definitions" and that the auth
model should be revisited when looking at ReBAC (relationship-based
access control). For now, the per-method scope annotation is the
authoritative way to mark a method as auth-required or public.

## Proto Regeneration

Generated code under `api/gen/` is gitignored (see `.gitignore`, under
"Generated API Files"). To regenerate it locally, run `task
protobuf/generate`, which runs `buf generate` from the `api/`
directory. The `buf.lock` pins the tool versions; do not commit a
diff in the lockfile unless you have intentionally upgraded `buf`.

## Observability

The server emits OpenTelemetry traces and metrics. The pipeline is
initialised in `otel.Init`, called from `cmd/serve.go::RunServe` before
the server starts listening. The OTel SDK is configured via the existing
`cfg/cfg.go` flag-set pattern plus standard OTel SDK env vars
(`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, etc.).

### What gets emitted

* **Span names** are business-operation names defined in `AGENTS.md`:
  `create_link`, `resolve_link`, `redirect`, `storage.lookup`,
  `storage.write`. These are emitted via the OTel auto-instrumentation
  libraries (`otelhttp`, `otelgrpc`) with span-name formatters that
  rename the default method/path names.
* **Custom business metrics** under the `x40.link` namespace:
  `links.created`, `links.resolved`, `links.not_found`,
  `storage.errors`. The first three carry a `storage` label
  (`boltdb`, `hashmap`, `yaml`, `firestore`) and a `surface` label
  (`http`, `grpc`); the last carries `storage` and `op` (`lookup`,
  `write`).
* **Go runtime metrics** (`process.runtime.go.*`) and host metrics.
* **HTTP / gRPC semantic-convention metrics** (`http.server.request.duration`,
  `rpc.server.duration`, etc.) from the contrib libraries.

### Local development

The default OTLP endpoint is `cloudtrace.googleapis.com:4317`. To send
data to a local collector instead, set the standard OTel env var:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 ./x40.link serve \
    --storage.boltdb.file /tmp/x40.link.db
```

A runnable collector config is in `docs/content/how-to/observe-locally.md`.

### Production (Cloud Run)

The Cloud Run service account needs two IAM roles:

* `roles/cloudtrace.agent` — write traces to Cloud Trace.
* `roles/monitoring.metricWriter` — write metrics to Cloud Monitoring.

Application Default Credentials (via the Cloud Run metadata server)
authenticate the OTel exporter against these endpoints; no service
account key file is required inside the container.

The OTLP exporter dials over **IPv4 only**. Cloud Run's network
configuration can resolve `cloudtrace.googleapis.com` to an IPv6
address and time out before the IPv4 fallback completes, causing
the metric export to fail with `i/o timeout`. Forcing IPv4 at the
dialer level avoids the IPv6 path entirely. See `otel/ipv4_dialer.go`
(or the `ipv4Dialer` function in `otel/init.go`) for the implementation;
no configuration is required — the dialer is unconditionally applied.

The `service.version` resource attribute is set from `version.Version`,
which is overridden at link time via:

```
go build -ldflags "-X github.com/andrewhowdencom/x40.link/version.Version=<value>" main.go
```

The `task bin/*` build commands and the `Containerfile` both inject this
flag. When unset, the value is `"unknown"`.

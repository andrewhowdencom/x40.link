# Plan: Add `resolve` Subcommand to the x40.link CLI

## Objective
Add a `x40 resolve <short-url>` subcommand to the CLI that prints the destination URL a short link points to. The command must work end-to-end without OAuth credentials — both the CLI does not require the user to authenticate, and the server-side `Get` RPC must be callable without a token. Existing `x40 <url>` (create-short-link) behavior must be preserved.

## Context

**Repository layout (relevant to this plan):**
- `cli/main.go` — the CLI binary. Today a single `cobra.Command` with `RunE: DoURL`; takes a destination URL and creates a short link. Inherits a flag set `urlFlagSet` that mixes `cfg.APIEndpoint` with several `cfg.OAuth2*` entries.
- `cli/auth/auth.go` — `TokenSource()` constructs an `oauth2.TokenSource` from Viper config, including `oauth2.Config` populated with `api.X40PermissionsList()`.
- `cli/auth/per_rpc_creds.go` — `NewPerRPCCredentials(ts)` wraps a `TokenSource` into a `grpc.PerRPCCredentials` that injects `Authorization: Bearer <token>` on every call.
- `api/api.go` — `X40Permissions()` reads the `oauth2_scope` proto extension from every method of every service in the `x40.dev.url` and `x40.dev.auth` packages, returning a `method → scope` map. `NewGRPCClient(addr, opts...)` accepts variadic `grpc.DialOption`, so we can simply *not* pass per-RPC credentials for unauthenticated calls.
- `api/dev/url.proto` — defines `ManageURLs` service with `Get` and `New` RPCs. Both currently carry an `oauth2_scope` annotation. The `Response` message is `{ string url = 1; }`.
- `api/dev/url.go` — server implementation. `Get` calls `Storer.Get(ctx, url)`, returns `codes.NotFound` / `codes.PermissionDenied` / `codes.InvalidArgument` / `codes.Internal` per the storage error. No instrumentation today.
- `api/auth/jwts/server_interceptor.go` — `ValidateCtx` enforces the scope on every gRPC call. It uses `o.Permissions[method]`; if the method has no entry, it returns `auth.ErrCannotAuthorize` ("no scope for the method"). If the entry exists, it requires the `Authorization` header and parses the JWT.
- `api/di/viper.go` — `OptsFromViper()` registers the JWT interceptor as a soft dependency; if `jwts.WireServerInterceptor()` fails, the server runs *without* auth enforcement.
- `storage/storage.go` — `Storer` interface: `Get(ctx, *url.URL) (*url.URL, error)` and `Put(ctx, from, *url.URL, to *url.URL) error`. Errors include `ErrNotFound`, `ErrUnauthorized`, `ErrFailed`, `ErrCorrupt`, etc.
- `server/storage.go` — `strHandler.Redirect` calls `Storer.Get` directly, *not* the gRPC `Get` RPC. So the gRPC `Get` is purely a client-facing API and is not on the redirect hot path.
- `cfg/cfg.go` — flag definitions and Viper binding. `cfg.APIEndpoint` is the only flag the `resolve` subcommand will need.
- `Taskfile.yml` — standard tasks: `test` runs `go test ./... -test.v -race -vet=all`; `protobuf/generate` runs `cd api && buf generate`.

**Project conventions** (from `AGENTS.md`):
- Test-driven development: write tests for current behavior, modify them to assert the new behavior, implement, validate.
- Linux-kernel-style commit messages.
- Add OpenTelemetry instrumentation when changing business logic or adding RPCs. (Flagged in the plan; not blocking for this effort, see Task 1.)
- Update README/DEVELOPMENT/docs after implementation.

## Architectural Blueprint

**Selected architecture: extend the existing CLI to add a subcommand; make the existing `Get` RPC public at the protocol level.**

The server already implements `Get`. The gRPC mux already routes it. The CLI just needs a new subcommand. The only nontrivial work is the auth model on `Get`: the proto's `oauth2_scope` annotation on `Get` is currently enforced by the JWT interceptor, so we must:

1. Remove the scope annotation from `Get` in `api/dev/url.proto`.
2. Update `api/auth/jwts/server_interceptor.go::ValidateCtx` to treat a method with an empty (or absent) scope as public — i.e., skip the JWT validation step but still allow the request through with an unauthenticated context.

This is consistent with the data's actual sensitivity: the destination of a short link is functionally public, since the HTTP redirect already discloses it to anonymous users. The HTTP redirect handler (`server/storage.go::Redirect`) does not go through the gRPC `Get` RPC and is not affected by this change.

**Tree-of-Thought deliberation (alternatives considered):**
- *Path B — add a new unauthenticated `Resolve` RPC alongside `Get`.* Rejected: duplicates the conceptual operation, grows the proto surface, complicates the docs. The information-theoretic content of `Get` is not sensitive.
- *Path C — add a new `unauthenticated` proto extension and have the interceptor respect it.* Rejected: more elaborate than A; same end result; not justified at this scale.

**Components and their interactions:**
- User runs `x40 resolve https://dhse.link/$`.
- Cobra dispatches to `resolveCmd` (new), which calls `DoResolve`.
- `DoResolve` parses the URL, calls `api.NewGRPCClient(cfg.APIEndpoint)` with no per-RPC credentials, then `client.Get(ctx, &gendev.GetRequest{Url: inputUrl})`.
- The gRPC server receives the call. With the new interceptor behavior, the empty-scope `Get` method passes through without auth. The `URL.Get` handler in `api/dev/url.go` looks up the short URL in storage and returns the destination.
- The CLI prints the destination to stdout.

## Requirements

1. A new CLI subcommand `x40 resolve <short-url>` that prints the destination URL.
2. The subcommand must not require OAuth configuration or authentication on the CLI side.
3. The subcommand must not require authentication on the server side; the `Get` RPC must be callable without credentials.
4. The existing `x40 <url>` behavior (create a short link) must be preserved unchanged.
5. The subcommand must surface `codes.NotFound` ("URL not found") and `codes.InvalidArgument` ("URL parse failure") as user-friendly errors on stderr with the appropriate `sysexits` code; other gRPC errors become a generic protocol error.
6. Input URLs without a scheme default to `https://`, matching the existing `New` behavior.
7. Unit tests, written table-driven and parallel, must cover: valid input → success; storage not found → `sysexits.DataErr`-style exit; bad URL parse → `sysexits.DataErr`-style exit; gRPC transport error → `sysexits.NoHost` or `sysexits.Protocol` style exit. `[inferred]`
8. No new dependencies in `go.mod` are introduced. `[inferred]`
9. Telemetry: the existing `Get` handler gains no tracing. The interceptor change is untraced. (Flagged as a follow-up: this is a small change and the existing RPCs are not instrumented; adding OTel here without doing the same for `New` would create inconsistency. A separate effort should instrument both.) `[inferred from AGENTS.md]`

## Task Breakdown

### Task 1: Make the `Get` RPC publicly callable
- **Goal**: Remove the OAuth scope requirement on the `Get` RPC so that the CLI can call it without sending credentials.
- **Dependencies**: None.
- **Files Affected**:
  - `api/dev/url.proto` — remove the `option (x40.dev.auth.oauth2_scope) = "..."` line on the `Get` RPC. Keep the `New` RPC's scope annotation intact.
  - `api/gen/dev/url.pb.go` and `api/gen/dev/url_grpc.pb.go` — regenerated by `buf generate`. Do not hand-edit; commit them with the proto change.
  - `api/auth/jwts/server_interceptor.go` — modify `ValidateCtx` to allow methods whose scope is empty (or whose `o.Permissions[method]` entry is the empty string). The cleanest implementation: after the `o.Permissions[method]` lookup, check if the scope is `""`; if so, return the context unchanged and a `nil` error, and *do not* require the `Authorization` header. (Make sure the new code path doesn't reject subsequent legitimate calls.)
  - `api/auth/jwts/server_interceptor_test.go` — add a new table-driven test case to `TestValidateCtx` (or equivalent test function): a method present in `Permissions` with an empty scope passes through without metadata, with no `Authorization` header, and does not set `CtxKeyAgent` on the context.
- **New Files**: None.
- **Interfaces**:
  - `api/dev/url.proto::ManageURLs.Get` — loses the `oauth2_scope` extension. Behavior at the gRPC layer changes from "requires scope `api.x40.link/scopes/x40.dev.url.ManageURLs.Get`" to "no auth required."
- **Validation** (run all before commit):
  - `cd api && buf generate` succeeds without diff to `buf.lock`.
  - `go build ./...` is clean.
  - `task test` (i.e., `go test ./... -test.v -race -vet=all`) is green.
  - The new interceptor test case passes; existing interceptor test cases (which test the auth-required path on `New`) still pass.
  - Manually: with the server running and a known short link in storage, a `grpcurl -plaintext -d '{"url": "https://example.com/abc"}' localhost:8080 x40.dev.url.ManageURLs/Get` (or the TLS equivalent) returns a `Response` without needing a token. A `New` call still requires a token. (This smoke test is the implementer's responsibility and is not a CI gate.)
- **Details**:
  - TDD order: write the new interceptor test case first, watch it fail (or pass under the new desired behavior), then implement the change in `ValidateCtx`. The proto change should follow — regenerate, verify the generated Go reflects the change.
  - Commit the proto, the generated code, and the interceptor change together so the repository remains in a buildable state at every commit.
  - No instrumentation is added in this task. See Requirements #9 for rationale.

### Task 2: Add `resolve` subcommand to the CLI
- **Goal**: Add a `x40 resolve <short-url>` subcommand that prints the destination URL, with no auth required and no breaking changes to existing CLI behavior.
- **Dependencies**: Task 1.
- **Files Affected**:
  - `cli/main.go` — substantial refactor:
    - Split `urlFlagSet` into two flag sets: `apiFlagSet` (containing only `cfg.APIEndpoint`) and `authFlagSet` (the `cfg.OAuth2*` entries).
    - The existing `Root` (the `New` command) uses a combined flag set (`apiFlagSet` + `authFlagSet`) to preserve current behavior.
    - Add a new `resolveCmd` (a `*cobra.Command` with `Use: "resolve"`, `Args: cobra.ExactArgs(1)`, `RunE: DoResolve`) that uses only `apiFlagSet`.
    - Register `resolveCmd` via `Root.AddCommand(resolveCmd)`.
    - Implement `DoResolve(_ *cobra.Command, args []string) error`:
      1. Parse `args[0]`, prepending `https://` if no scheme is present (mirror the existing logic in `DoURL`).
      2. Build the gRPC client: `client, err := api.NewGRPCClient(viper.GetString(cfg.APIEndpoint.Path))`. Note: *no* per-RPC credentials.
      3. Call `client.Get(ctx, &gendev.GetRequest{Url: input})` with a context timeout (suggest 10s, matching `DoURL`).
      4. On `codes.NotFound`, return a `sysexits.DataErr`-style wrapped error and print "url not found: <url>" to stderr.
      5. On `codes.InvalidArgument`, return a `sysexits.DataErr`-style wrapped error and print "invalid url: <url>" to stderr.
      6. On any other gRPC error, return a `sysexits.Protocol` (or `sysexits.Software`) wrapped error.
      7. On success, print the destination URL to stdout (no prefix, no decoration, similar to how `DoURL` prints the short URL).
    - Update the `Root.Long` description to mention the new subcommand and show usage for both forms.
  - `cli/main_test.go` (new file) — table-driven tests for `DoResolve`. Cover:
    - Valid input URL → prints the destination to a captured stdout.
    - Input URL with no scheme → defaults to `https://`.
    - `codes.NotFound` from the server → exit-style error wrapping `storage.ErrNotFound` semantics.
    - `codes.InvalidArgument` from the server → exit-style error.
    - Transport-level failure (e.g., `NewGRPCClient` returns an error) → exit-style error.
    - Each test case sets up a fake gRPC server (or a fake `Client`) returning the desired response. The simplest approach is to define a small interface in the test file that `DoResolve` consumes, and have the test inject a fake. *This requires a small refactor in `DoResolve` to accept the client as a parameter or via a package-level seam — flagged below.*
- **New Files**:
  - `cli/main_test.go` — new test file.
- **Interfaces**:
  - `cli/main.go::resolveCmd` — new `*cobra.Command`.
  - `cli/main.go::DoResolve` — new function. Suggested signature: `func DoResolve(_ *cobra.Command, args []string) error`. To make it testable, factor out a `doResolveWithClient(ctx context.Context, client api.Client, input string) (string, error)` helper that takes the client and the input string. The `DoResolve` cobra function then handles argument parsing, client construction, timeout context, and printing. Test cases drive `doResolveWithClient` directly.
- **Validation** (run all before commit):
  - `go build ./...` is clean.
  - `go vet ./...` is clean.
  - `go test ./cli/... -race` is green. The new tests in `cli/main_test.go` pass; the existing CLI behavior (if any existing tests) is unaffected.
  - `task test` is green overall.
  - Manually: with the server running, `x40 resolve https://x40.link/abc` prints the destination. `x40 https://dest.example/` still creates a short link (existing behavior preserved). `x40 --help` lists the `resolve` subcommand and shows its usage.
- **Details**:
  - The exact `sysexits` codes for each error path are a planner decision; the test cases should assert the wrapping is consistent (e.g., `errors.Is` against `sysexits.DataErr`).
  - Refactor the flag set carefully: do *not* rename `urlFlagSet` in a way that breaks any other consumers; the simplest path is to keep `urlFlagSet` as-is and add the two new sets alongside it, with `urlFlagSet` becoming a composition of `apiFlagSet` and `authFlagSet`.
  - Commit order: refactor (flag set split, no behavior change), then add `DoResolve` and `resolveCmd` with tests, then update the help text. Each commit should leave the repository in a buildable, tested state.

### Task 3: Update documentation
- **Goal**: Reflect the new `resolve` subcommand in user-facing and developer-facing documentation.
- **Dependencies**: Task 2.
- **Files Affected**:
  - `README.md` — add a brief mention of the `resolve` subcommand in the "Usage" section (or a new "Inspecting" subsection). One or two sentences plus a usage example.
  - `DEVELOPMENT.md` — note that the CLI now has a subcommand, and document the auth model change (i.e., that `Get` is public). One short paragraph.
  - `docs/` — the docs site. Identify the relevant page (likely the "Using the CLI" or "API" page) and add a section on `resolve`. If the docs are in a separate submodule or build pipeline, flag this to the implementer; it may need a follow-up.
- **New Files**: None, unless the docs site requires new page files.
- **Interfaces**: No code changes.
- **Validation** (run all before commit):
  - The CLI's `--help` output matches the new `Long` description.
  - `task docs` (or whatever builds the docs site) succeeds if applicable.
  - Manual review: a first-time reader can find out how to resolve a short URL from the README.
- **Details**:
  - Keep documentation changes tightly scoped; do not rewrite surrounding prose.
  - If the docs site build is heavy or external, treat that as a separate commit/PR and skip it in this effort (the README + DEVELOPMENT updates are the minimum).

## Dependency Graph

- Task 1 → Task 2 (the CLI subcommand can only be useful once the server accepts unauthenticated `Get` calls)
- Task 2 → Task 3 (docs describe the CLI surface)

No parallelizable tasks within this scope. The work is small and sequential.

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|
| The proto regeneration (`buf generate`) produces a large diff in `api/gen/dev/`. Reviewers may be uneasy. | Low (generated code, not hand-written) | High | Include the full generated diff in the commit message with a clear "regenerated" note. The `buf.lock` pins the tool versions. |
| Making `Get` public is a security-posture change. A server admin who relied on auth on `Get` may be surprised. | Medium | Low | Document the change in `DEVELOPMENT.md` (Task 3) and in the commit message. The data is functionally public (HTTP redirect discloses the same info); this aligns the gRPC API with that reality. |
| Cobra subcommand dispatch may not behave as expected given `Root.Args = cobra.MinimumNArgs(1)`. | Medium (could break existing CLI) | Low | Smoke-test both `x40 <url>` and `x40 resolve <url>` manually before committing Task 2. If the existing root args conflict, refactor `Root` to a parent command and have `newCmd` and `resolveCmd` be siblings. |
| The `cli/main.go::DoURL` always calls `auth.TokenSource()`, which depends on Viper config. If `resolve` accidentally invokes this path, the user gets a confusing "no oauth2 client id configured" error. | Medium (UX) | Medium | Carefully refactor so that `DoResolve` does *not* call `auth.TokenSource()` or any of the auth code paths. Unit tests should cover the case where OAuth config is empty. |
| Adding OTel to the interceptor or the `Get` handler is consistent with `AGENTS.md` "instrumentation" guidance but would create asymmetry with `New` (which is also uninstrumented). | Low | Medium | Defer OTel to a separate effort that instruments both `Get` and `New` together. Note this explicitly in the Task 1 commit message. |
| The JWT interceptor's `ValidateCtx` change is subtle (allow empty-scope methods). A bug here could weaken auth on other methods. | High (security) | Low | Add explicit unit tests for the empty-scope case *and* confirm existing tests for the auth-required case still pass. Make the change minimal: a single early-return when the looked-up scope is `""`. |
| `doResolveWithClient` testability refactor changes the call signature; could ripple if misused. | Low | Low | Keep the helper unexported. Document the seam in a comment. |

## Validation Criteria

- [ ] `cd api && buf generate` produces no diff to `buf.lock` and the generated `url.pb.go` no longer has the scope extension on `Get`.
- [ ] `go build ./...` succeeds.
- [ ] `task test` (i.e., `go test ./... -test.v -race -vet=all`) is green.
- [ ] `go vet ./...` is clean.
- [ ] A new unit test in `api/auth/jwts/server_interceptor_test.go` confirms that an empty-scope method passes through `ValidateCtx` without metadata or an `Authorization` header.
- [ ] All existing interceptor unit tests still pass (i.e., `New` still requires the scope).
- [ ] A new unit test in `cli/main_test.go` confirms `doResolveWithClient` returns the destination for a valid response, surfaces `codes.NotFound` as a wrapped error, and surfaces `codes.InvalidArgument` as a wrapped error.
- [ ] `x40 --help` lists the `resolve` subcommand.
- [ ] `x40 resolve https://x40.link/<known-slug>` prints the destination URL when run against a server with that link in storage.
- [ ] `x40 <url>` still creates a short link and prints the short URL (existing behavior preserved).
- [ ] `x40 resolve` does not require any OAuth-related flags or environment variables to be set.
- [ ] `README.md` and `DEVELOPMENT.md` mention the new subcommand and the public `Get` RPC posture, respectively.
- [ ] The repository is in a committable, buildable, tested state after each task.

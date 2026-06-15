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

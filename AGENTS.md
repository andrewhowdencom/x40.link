# Agents

## Git Commits

For git commit messsages, be sure to write them in a format that follows the Linux kernel commit conventions. For example, break at 72 characters, 50 line title and so on. Be sure to include the context of the change, including:

* Why you made the change
* What design notes are useful for future engineers. For any changes that modify public APIs (both in and out of process), be sure to note that there is a breaking change in this commit message by following the syntax:

BREAKING CHANGE: <short-summary-of-api>
  <steps-to-migrate-between-the-old-and-new-api>

Commit meessages should generally be of the form:

```
This is the title of the commit message

This is the body of the commit messsage, in which you
will describe the different things that you changed, and
why you changed them.

== Design Notes
=== Specific Callout

This is a specific callout for a design choice that you
why you made it, what alternatives existed and why this
is superior.
```

## Instrumentation

Where making changes to the business logic, adding new RPCs or similar be sure to ensure that there is appropriate telemetry (via OpenTelemetry) that allows validating those changes are working correctly in production. As a rule, prefer distributed tracing over metrics, and metrics over logs. In all cases, be sure to follow the semantic conventions.

### Traces

Give the traces appropriate operation names that reflect the the business function (e.g. "create_link"), rather than an application artifact (e.g. HTTP POST /link/create). Do not duplicate traces for "business operations" versus "HTTP requests", but rather, if there's a span reflecting the HTTP request already, modify its name and attributes to match the "business operation".

## Tests

For just about all non-trivial changes, make sure you develop via "test driven design". This means:

1. Write tests for the _current_ behavior of the application
2. Modify those tests so that they verify the _new desired_ behavior of the application 
3. Modify the logic of the application based on your request, so it validates against those tests.
4. Adjust either the application or the tests until the tests pass
5. Publish the change.


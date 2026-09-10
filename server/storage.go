package server

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/andrewhowdencom/x40.link/otel"
	"github.com/andrewhowdencom/x40.link/server/message"
	"github.com/andrewhowdencom/x40.link/storage"
	"go.opentelemetry.io/otel/trace"
	"schneider.vip/problem"
)

type strHandler struct {
	str storage.Storer
}

// Redirect receives a request, and if it matches a storage, responds.
//
// On success, the handler writes a 307 Temporary Redirect to the
// resolved destination. On failure, it writes the RFC 7807 problem
// document directly to the response — it does not depend on the
// server.Error middleware to do so, because the otelhttp middleware
// installed by server.WithOtel creates a new request to propagate
// trace context, which means the WithError mutation mechanism (used
// by older code in this codebase) does not propagate past the
// otelhttp boundary.
func (o *strHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	lookup := &url.URL{
		Host: r.Host,
		Path: r.URL.Path,
	}

	red, err := o.str.Get(r.Context(), lookup)

	if errors.Is(err, storage.ErrNotFound) {
		writeProblem(w, r, problem.New(
			problem.Status(http.StatusNotFound),
			problem.Custom("url", lookup.String()),
			problem.WrapSilent(err),
		))
		return
	}

	if err == nil {
		w.Header().Add("Location", red.String())
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	writeProblem(w, r, problem.New(
		problem.Status(http.StatusInternalServerError),
		problem.WrapSilent(err),
	))
}

// writeProblem serialises an RFC 7807 problem document to w in the
// format the client prefers (XML or JSON, default JSON).
func writeProblem(w http.ResponseWriter, r *http.Request, p *problem.Problem) {
	switch r.Header.Get(message.HeaderAccept) {
	case message.MIMETextXML, message.MIMEApplicationXML:
		_, _ = p.WriteXMLTo(w)
	default:
		_, _ = p.WriteTo(w)
	}
}

// instrumentedRedirect wraps a strHandler with custom business metrics.
// It deliberately does not start a new span — the otelhttp middleware
// (added via server.WithOtel) is already responsible for the "redirect"
// span. This wrapper is only the metric-emission seam.
type instrumentedRedirect struct {
	inner   *strHandler
	storage string
}

// Redirect delegates to the inner handler, then emits the appropriate
// business counter based on the response status code written by the
// inner handler.
func (i *instrumentedRedirect) Redirect(w http.ResponseWriter, r *http.Request) {
	// Touch the span from context so the OTel no-op span is exercised
	// on the no-tracer code path; this is a no-op when no tracer is
	// configured.
	trace.SpanFromContext(r.Context())

	// Wrap the response writer so we can observe the status code
	// that the inner handler chose to write, without re-implementing
	// the redirect logic.
	rw := &statusCapturingWriter{ResponseWriter: w}
	i.inner.Redirect(rw, r)

	switch rw.status {
	case 0, http.StatusTemporaryRedirect, http.StatusOK:
		// No status set, or a successful status. strHandler.Redirect
		// leaves the status at 0 when it calls WriteHeader, so the
		// default is to count the request as a resolve.
		otel.RecordResolved(i.storage, otel.SurfaceHTTP)
	case http.StatusNotFound:
		otel.RecordNotFound(i.storage, otel.SurfaceHTTP)
	default:
		// Any other status (e.g. 5xx from a storage failure) is a
		// storage error.
		otel.RecordStorageError(i.storage, otel.OpLookup)
	}
}

// statusCapturingWriter wraps an http.ResponseWriter and records
// the status code that was written so the wrapper can react after
// the inner handler returns.
type statusCapturingWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCapturingWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.status = http.StatusOK
		w.wrote = true
	}
	return w.ResponseWriter.Write(b)
}

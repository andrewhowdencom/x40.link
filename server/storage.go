package server

import (
	"net/http"
	"net/url"

	"github.com/andrewhowdencom/x40.link/storage"
	"schneider.vip/problem"
)

type strHandler struct {
	str storage.Storer
}

// Redirect receives a request, and if it matches a storage, responds.
func (o *strHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	lookup := &url.URL{
		Host: r.Host,
		Path: r.URL.Path,
	}

	red, err := o.str.Get(lookup)

	if err == storage.ErrNotFound {
		WithError(r, problem.New(
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

	WithError(r, problem.New(
		problem.Status(http.StatusInternalServerError),
		problem.WrapSilent(err),
	))
}

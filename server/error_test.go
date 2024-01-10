package server_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrewhowdencom/x40.link/server"
	"github.com/andrewhowdencom/x40.link/server/message"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"schneider.vip/problem"
)

func TestErrorMiddleware(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		accept  string

		status  int
		body    []byte
		recover any
	}{
		{
			name: "has no errors",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Yeah!"))
			},

			status: http.StatusOK,
			body:   []byte("Yeah!"),
		},
		{
			name: "has a problem error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				server.WithError(r, problem.New(
					problem.Status(http.StatusConflict),
				))
			},

			status: http.StatusConflict,
			body:   []byte(problem.New(problem.Status(http.StatusConflict)).JSON()),
		},
		{
			name: "has an unknown error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				server.WithError(r, errors.New("Not sure what happened here"))
			},

			status: http.StatusInternalServerError,
			body:   []byte(server.ProblemUnknown.JSON()),
		},
		{
			name: "has non error type",
			handler: func(w http.ResponseWriter, r *http.Request) {
				*r = *r.WithContext(
					context.WithValue(r.Context(), server.CtxErrors, "Quack quack"),
				)
			},
			recover: "a non-error type added as error context",
		},
		{
			name: "has json content type",
			handler: func(w http.ResponseWriter, r *http.Request) {
				server.WithError(r, problem.New(
					problem.Status(http.StatusConflict),
				))
			},
			accept: message.MIMEApplicationJSON,
			status: http.StatusConflict,
			body:   []byte(problem.New(problem.Status(http.StatusConflict)).JSON()),
		},
		{
			name: "has application/xml content type",
			handler: func(w http.ResponseWriter, r *http.Request) {
				server.WithError(r, problem.New(
					problem.Status(http.StatusConflict),
				))
			},
			accept: message.MIMEApplicationXML,
			status: http.StatusConflict,
			body:   []byte(problem.New(problem.Status(http.StatusConflict)).XML()),
		},
		{
			name: "has text/xml content type",
			handler: func(w http.ResponseWriter, r *http.Request) {
				server.WithError(r, problem.New(
					problem.Status(http.StatusConflict),
				))
			},
			accept: message.MIMETextXML,
			status: http.StatusConflict,
			body:   []byte(problem.New(problem.Status(http.StatusConflict)).XML()),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				assert.Equal(t, tc.recover, recover())
			}()

			t.Parallel()

			mux := chi.NewMux()
			mux.Use(server.Error)
			mux.Get("/", tc.handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			req.Header.Add(message.Accept, tc.accept)

			mux.ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Result().StatusCode)
			assert.Equal(t, tc.body, w.Body.Bytes())
		})
	}
}

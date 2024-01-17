// Package dev implements a GRPC server that reads and writes URLs to storage.
package dev

import (
	"context"
	"errors"
	"log"
	"net/url"

	"github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/uid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// URLEnricher are defaults applied when the user doesn't supply that information.
type URLEnricher struct {
	Domain string
	Path   *uid.Generator
}

// Enrich adds information to the provided URL if it is not already present.
func (u *URLEnricher) Enrich(from *url.URL, to *url.URL) error {
	if from.Host == "" {
		from.Host = u.Domain
	}

	if from.Path != "" {
		return nil
	}

	id, err := u.Path.ID(to)
	if err != nil {
		return err
	}

	from.Path = "/" + id

	return nil
}

// URL is an implementation of the URL gRPC Server
type URL struct {
	Storer storage.Storer

	Enricher func(from *url.URL, to *url.URL) error

	dev.UnimplementedManageURLsServer
}

// Get fetches a URL from storage
func (u URL) Get(ctx context.Context, req *dev.GetRequest) (*dev.Response, error) {
	url, err := url.Parse(req.Url)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "url parse failure: %s", err)
	}

	response, err := u.Storer.Get(ctx, url)
	if errors.Is(err, storage.ErrUnauthorized) {
		return nil, status.Error(codes.PermissionDenied, "you are not the owner of this record")
	} else if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "url not found")
	} else if err != nil {

		log.Println(err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &dev.Response{
		Url: response.String(),
	}, nil
}

// New generates the "from" URL on the fly
func (u URL) New(ctx context.Context, req *dev.NewRequest) (*dev.Response, error) {
	to, err := url.Parse(req.SendTo)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "url parse failure: %s", err)
	}

	from := &url.URL{}
	if req.On != nil {
		from.Host = req.On.Host
		from.Path = req.On.Path
	}

	// Add information if it is not there.
	if err := u.Enricher(from, to); err != nil {
		log.Println(err)
		return nil, status.Error(codes.Internal, "unable to add missing information")
	}

	err = u.Storer.Put(ctx, from, to)

	if errors.Is(err, storage.ErrUnauthorized) {
		return nil, status.Error(codes.PermissionDenied, "you are not the owner of this record")
	} else if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to write to storage")
	}

	return &dev.Response{
		Url: from.String(),
	}, nil
}

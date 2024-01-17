// Package dev implements a GRPC server that reads and writes URLs to storage.
package dev

import (
	"context"
	"errors"
	"log"
	"net/url"

	"github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// URL is an implementation of the URL gRPC Server
type URL struct {
	Storer storage.Storer

	dev.UnimplementedManageURLsServer
}

// Get fetches a URL from storage
func (u URL) Get(ctx context.Context, req *dev.Request) (*dev.Response, error) {
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

// New generates the "from" URL on the fly, based on request metadata.
func (URL) New(context.Context, *dev.Request) (*dev.Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Post not implemented")
}

// NewCustom writes a new URL into storage
func (u URL) NewCustom(ctx context.Context, req *dev.CustomRequest) (*emptypb.Empty, error) {
	urls := [2]*url.URL{}

	for i, v := range []string{req.From, req.To} {
		url, err := url.Parse(v)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "url parse failure: %s", err)
		}

		urls[i] = url
	}

	err := u.Storer.Put(ctx, urls[0], urls[1])

	if errors.Is(err, storage.ErrUnauthorized) {
		return nil, status.Error(codes.PermissionDenied, "you are not the owner of this record")
	} else if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to write to storage")
	}

	return &emptypb.Empty{}, nil
}

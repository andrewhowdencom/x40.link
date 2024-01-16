// Package firestore implements a storage layer with Google cloud firestore.
package firestore

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/andrewhowdencom/x40.link/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// document is the internal format for the data stored in firestore
type document struct {
	// To is where the url should be sent
	To string `firestore:"to"`

	// Owner is the owner of the document
	Owner string `firestore:"owner"`
}

// FirestoreCollection is the collection (in practice, path prefix) for accessing URL content.
const FirestoreCollection = "links"

// Firestore is the implementation of Google Cloud firestore backed storage
type Firestore struct {
	Client *firestore.Client
}

// Get fetches a URL from storage
func (fs Firestore) Get(_ context.Context, url *url.URL) (*url.URL, error) {
	ref := fs.Client.Doc(urlToPath(url))
	doc, err := fs.doc(ref)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: %s", storage.ErrNotFound, "data at path not found")
	} else if err != nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrFailed, err)
	}

	to, err := url.Parse(doc.To)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrCorrupt, err)
	}

	return to, nil
}

// Put writes a URL into storage
// TODO: Write tests for all this.
func (fs Firestore) Put(ctx context.Context, from *url.URL, to *url.URL) error {
	ref := fs.Client.Doc(urlToPath(from))
	agent, _ := ctx.Value(storage.CtxKeyAgent).(string)

	// Fetch the owner from the request (if there is one)
	owner := ""
	val, ok := ctx.Value(storage.CtxKeyAgent).(string)
	if ok && val != "" {
		owner = val
	}

	// See if there is a document already, and if so, see who owns it.
	doc, err := fs.doc(ref)
	status, _ := status.FromError(err)

	if status.Code() != codes.OK && status.Code() != codes.NotFound {
		return fmt.Errorf("%w: %s", storage.ErrFailed, err)
	}

	if status.Code() != codes.NotFound && doc.Owner != agent {
		return storage.ErrUnauthorized
	}

	// Try and create the document
	_, err = ref.Set(context.Background(), document{
		To:    to.String(),
		Owner: owner,
	})

	if err != nil {
		return err
	}

	return nil
}

// Owns implements the interface validating whether a user actually owns this record.
func (fs Firestore) Owns(ctx context.Context, u *url.URL) bool {
	// See who is requesting this data
	agent, ok := ctx.Value(storage.CtxKeyAgent).(string)
	if !ok || agent == "" {
		return false
	}

	ref := fs.Client.Doc(urlToPath(u))
	if ref == nil {
		return false
	}

	doc, err := fs.doc(ref)
	if err != nil {
		return false
	}

	return doc.Owner == agent
}

func (fs Firestore) doc(ref *firestore.DocumentRef) (*document, error) {
	ctx, cxl := context.WithTimeout(context.Background(), time.Second*30)
	defer cxl()

	doc, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	result := &document{}
	if err := doc.DataTo(result); err != nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrCorrupt, err)
	}

	return result, nil
}

// urlToPath converts the URL into a document ID.
func urlToPath(url *url.URL) string {
	p := []string{FirestoreCollection, url.Host}

	if url.Path != "" {
		p = append(p, "id", strings.Replace(url.Path, "/", "+", -1))
	}

	return path.Join(p...)
}

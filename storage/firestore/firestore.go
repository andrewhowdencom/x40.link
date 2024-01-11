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

// FirestoreCollection is the collection (in practice, path prefix) for accessing URL content.
const FirestoreCollection = "links"

// Firestore is the implementation of Google Cloud firestore backed storage
type Firestore struct {
	Client *firestore.Client
}

// Get fetches a URL from storage
func (fs Firestore) Get(url *url.URL) (*url.URL, error) {
	ctx, cxl := context.WithTimeout(context.Background(), time.Second*30)
	defer cxl()
	ref := fs.Client.Doc(urlToPath(url))
	if ref == nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrNotFound, "ref not found")
	}

	doc, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: %s", storage.ErrNotFound, "document not found")
	} else if err != nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrFailed, err)
	}

	data, err := doc.DataAt("to")
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: %s", storage.ErrNotFound, "data at path not found")
	} else if err != nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrFailed, err)
	}

	str, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("%w: received %t (expecting string)", storage.ErrCorrupt, data)
	}

	to, err := url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrCorrupt, err)
	}

	return to, nil
}

// Put writes a URL into storage
func (fs Firestore) Put(from *url.URL, to *url.URL) error {
	ref := fs.Client.Doc(urlToPath(from))

	// Try and create the document
	_, err := ref.Set(context.Background(), map[string]interface{}{
		"to": to.String(),
	})

	if err != nil {
		return fmt.Errorf("%w: %s", storage.ErrFailed, err)
	}

	return nil
}

// urlToPath converts the URL into a document ID.
func urlToPath(url *url.URL) string {
	p := []string{FirestoreCollection, url.Host}

	if url.Path != "" {
		p = append(p, "id", strings.Replace(url.Path, "/", "+", -1))
	}

	return path.Join(p...)
}

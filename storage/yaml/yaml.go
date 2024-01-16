package yaml

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"

	"github.com/andrewhowdencom/x40.link/storage"
	parser "gopkg.in/yaml.v3"
)

// Log is the logger for the library. Uses the default structured logger, but can be overridden to disable the output
// for this package.
var Log = slog.Default()

// Yaml is a simple, read only implement of storage that fetches its initial state from a file and then returns
// that state. It rejects any writes.
type yaml struct {
	str storage.Storer
}

// row is the implementation of the file format in YAML.
type row struct {
	// From is the source URL that will be redirected.
	From string `yaml:"from"`

	// To is the destination url To which the source will be redirected.
	To string `yaml:"to"`
}

// New generates the storer. It receives another storer which it will enrich with the content from the YAML,
// and an io.reader which is expected to supply the YAML (typically a file).
//
// Returns an error in the case there is a failure to store the URL or to wholely fail the YAML parsing, but
// ignores single line failures (simply skipping the record)
func New(str storage.Storer, src io.Reader) (*yaml, error) {
	y := &yaml{str: str}

	// Read the content into a structure that we can convert it to URLs
	rows := make([]row, 0)
	dec := parser.NewDecoder(src)
	err := dec.Decode(&rows)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", storage.ErrStorageSetupFailed, err)
	}

	// Fill up the storage with the links
	for _, r := range rows {
		from, err := url.Parse(r.From)
		if err != nil {
			Log.Debug("failed to parse from log", "url", r.From, "err", err)
			continue
		}

		to, err := url.Parse(r.To)
		if err != nil {
			Log.Debug("failed to parse from log", "url", r.To, "err", err)
			continue
		}

		fmt.Println(from.String(), to.String())

		if err := y.str.Put(context.Background(), from, to); err != nil {
			return nil, fmt.Errorf("%w: %s", storage.ErrStorageSetupFailed, err)
		}
	}

	return y, nil
}

func (y *yaml) Get(ctx context.Context, u *url.URL) (*url.URL, error) {
	return y.str.Get(ctx, u)
}

func (y *yaml) Put(context.Context, *url.URL, *url.URL) error {
	return storage.ErrReadOnlyStorage
}

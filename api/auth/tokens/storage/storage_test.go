package storage_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/andrewhowdencom/x40.link/api/auth/tokens/storage"
	"github.com/stretchr/testify/assert"
)

func TestStorageRW(t *testing.T) {
	// Create a new directory to use as the psuedo root for this package
	dir, err := os.MkdirTemp("", "test-storage-rw-*")
	if err != nil {
		t.Fatalf("unable to make test dir: %s", err.Error())
	}

	// Clean up that directory (and all its children)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("failed to clean up test dir: %s", err.Error())
		}
	}()

	// Give each subtest its own directory
	td := func(name string) string {
		dir, err := os.MkdirTemp(dir, strings.ReplaceAll(name, " ", "-"))
		if err != nil {
			t.Fatalf("failed to make test directory for %s: %s", name, err.Error())
		}

		return dir
	}

	for _, tc := range []struct {
		name string
		file *storage.File

		setup func(inDir string)

		// function list
		fl      []interface{}
		err     []error
		results [][]byte
	}{
		{
			name: "file not found",
			file: &storage.File{
				Path: "foo.json",
			},

			setup: func(name string) {},
			fl: []interface{}{
				func(f *storage.File) ([]byte, error) {
					return f.Read()
				},
			},
			err: []error{
				storage.ErrNotFound,
			},
			results: [][]byte{
				nil,
			},
		},
		{
			name: "file is directory",
			file: &storage.File{
				Path: "foo",
			},
			setup: func(inDir string) {
				if err := os.Mkdir(path.Join(inDir, "foo"), 0o755); err != nil {
					t.Fatalf("unable create test dir: %s", err.Error())
				}
			},
			fl: []interface{}{
				func(f *storage.File) ([]byte, error) {
					return f.Read()
				},
			},

			err: []error{
				storage.ErrStorageFailure,
			},
			results: [][]byte{
				nil,
			},
		},
		{
			name: "can write to path",
			file: &storage.File{
				Path: "foo.json",
			},
			setup: func(inDir string) {},
			fl: []interface{}{
				func(f *storage.File) error {
					return f.Write([]byte("Im going in the file!"))
				},
			},
			err: []error{
				nil,
			},
			results: [][]byte{},
		},
		{
			name: "can read back what was written",
			file: &storage.File{
				Path: "foo.json",
			},
			setup: func(inDir string) {},
			fl: []interface{}{
				func(f *storage.File) error {
					return f.Write([]byte("Im going in the file!"))
				},
				func(f *storage.File) ([]byte, error) {
					return f.Read()
				},
			},
			err:     []error{nil, nil},
			results: [][]byte{[]byte("Im going in the file!")},
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			dir := td(tc.name)
			tc.setup(dir)

			// Ensure that the path is relative to the newly created temporary directory.
			tc.file.Path = path.Join(dir, tc.file.Path)

			err := []error{}
			results := [][]byte{}

			// Run the function stack
			for _, f := range tc.fl {
				switch vf := f.(type) {
				case func(f *storage.File) ([]byte, error):
					r, e := vf(tc.file)
					err = append(err, e)
					results = append(results, r)
				case func(f *storage.File) error:
					err = append(err, vf(tc.file))
				default:
					t.Fatal("function supplied but not executable")
				}
			}

			// Compare the results
			for i := range tc.err {
				assert.ErrorIs(t, err[i], tc.err[i])
			}

			for i := range tc.results {
				assert.Equalf(t, tc.results[i], results[i], "byte construct not equal")
			}
		})
	}
}

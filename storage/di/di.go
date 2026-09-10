// Package di provides a mechanism to select storage providers. In a separate package so as to avoid an import cycle.
package di

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/boltdb"
	fsdb "github.com/andrewhowdencom/x40.link/storage/firestore"
	"github.com/andrewhowdencom/x40.link/storage/memory"
	"github.com/andrewhowdencom/x40.link/storage/yaml"
	"github.com/spf13/viper"
)

// Err* are sentinel errors
var (
	ErrCannotResolveStorage = errors.New("failed to find a possible storage configuration")
)

// WireStorage generates a storage engine from the Viper based configuration. Fails
// if there are no configuration values supplied.
//
// Returns the resolved Storer and a stable name identifying the
// backend ("hashmap", "yaml", "boltdb", "firestore"). The name is
// consumed by the server's instrumentation layer to label custom
// business counters.
//
// Doesn't actually use wire (yet)
//
// TODO: Rewrite this with the new configuration format.
func WireStorage() (storage.Storer, string, error) {
	if viper.GetBool(cfg.StorageHashMap.Path) {
		return memory.NewHashTable(), "hashmap", nil
	}

	if path := viper.GetString(cfg.StorageYamlFile.Path); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %s", ErrCannotResolveStorage, err)
		}

		y, err := yaml.New(memory.NewHashTable(), f)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %s", ErrCannotResolveStorage, err)
		}

		return y, "yaml", nil
	}

	if path := viper.GetString(cfg.StorageBoltDBFile.Path); path != "" {
		db, err := boltdb.New(path)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %s", ErrCannotResolveStorage, err)
		}

		return db, "boltdb", nil
	}

	if project := viper.GetString(cfg.StorageFirestoreProject.Path); project != "" {
		client, err := firestore.NewClient(context.Background(), project)

		if err != nil {
			return nil, "", fmt.Errorf("%w: %s", ErrCannotResolveStorage, err)
		}

		return fsdb.Firestore{
			Client: client,
		}, "firestore", nil
	}

	return nil, "", fmt.Errorf("%w: %s", ErrCannotResolveStorage, "no valid storage provider supplied")
}

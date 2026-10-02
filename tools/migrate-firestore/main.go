// migrate-firestore performs the offline migration to encoded source-path keys.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	store "github.com/andrewhowdencom/x40.link/storage/firestore"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	project := flag.String("project", "", "Firestore project (required)")
	apply := flag.Bool("apply", false, "apply the checked plan; application writers must be stopped")
	backup := flag.String("backup", "", "exclusive local backup file required with -apply (mode 0600)")
	rollback := flag.String("rollback", "", "restore a migration backup instead of migrating")
	flag.Parse()
	if *project == "" {
		return fmt.Errorf("-project is required")
	}
	if *apply && (*backup == "" || *rollback != "") {
		return fmt.Errorf("-apply requires -backup and cannot be combined with -rollback")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	client, err := firestore.NewClient(ctx, *project)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	fs := store.Firestore{Client: client}
	if *rollback != "" {
		f, err := os.Open(*rollback)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		var saved backupFile
		if err := json.NewDecoder(f).Decode(&saved); err != nil {
			return err
		}
		if saved.Project != *project {
			return fmt.Errorf("backup project does not match -project")
		}
		if err := fs.RollbackMigration(ctx, &saved.Plan); err != nil {
			return err
		}
		fmt.Println("rollback verified")
		return nil
	}
	plan, err := fs.PlanMigration(ctx)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(os.Stdout).Encode(plan.Summary()); err != nil {
		return err
	}
	if !*apply {
		return nil
	}
	if len(plan.Entries) == 0 {
		return fmt.Errorf("no legacy records to migrate")
	}
	f, err := os.OpenFile(*backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	encodeErr := json.NewEncoder(f).Encode(backupFile{Project: *project, Plan: *plan})
	syncErr := f.Sync()
	closeErr := f.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := fs.ApplyMigration(ctx, plan); err != nil {
		return fmt.Errorf("migration stopped; backup retained for rollback: %w", err)
	}
	fmt.Printf("migrated and verified %d records; backup: %s\n", len(plan.Entries), *backup)
	return nil
}

type backupFile struct {
	Project string              `json:"project"`
	Plan    store.MigrationPlan `json:"plan"`
}

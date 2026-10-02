package firestore

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MigrationRecord is a document snapshot included in the migration backup.
type MigrationRecord struct {
	Path       string                 `json:"path"`
	Data       map[string]interface{} `json:"data"`
	UpdateTime time.Time              `json:"update_time"`
}

// MigrationEntry describes one atomic move from a legacy key to an encoded key.
type MigrationEntry struct {
	Source     MigrationRecord        `json:"source"`
	Duplicates []MigrationRecord      `json:"duplicates,omitempty"`
	Target     string                 `json:"target"`
	Data       map[string]interface{} `json:"data"`
}

// MigrationPlan contains the backup and the complete, checked set of moves.
type MigrationPlan struct {
	Records             []MigrationRecord `json:"records"`
	Entries             []MigrationEntry  `json:"entries"`
	Placeholders        int               `json:"placeholders"`
	InferredPaths       int               `json:"inferred_paths"`
	AmbiguousKeys       int               `json:"ambiguous_keys"`
	RelativePaths       int               `json:"relative_paths"`
	EmptyDestinations   int               `json:"empty_destinations"`
	MissingOwners       int               `json:"missing_owners"`
	IdenticalDuplicates int               `json:"identical_duplicates"`
}

// Summary returns counts without exposing link destinations or owners.
func (p *MigrationPlan) Summary() map[string]int {
	return map[string]int{"moves": len(p.Entries), "placeholders": p.Placeholders, "inferred_paths": p.InferredPaths, "ambiguous_keys": p.AmbiguousKeys, "relative_paths": p.RelativePaths, "empty_destinations": p.EmptyDestinations, "missing_owners": p.MissingOwners, "identical_duplicates": p.IdenticalDuplicates}
}

// PlanMigration reads all legacy links and checks for target-key conflicts.
// It does not write to Firestore.
func (fs Firestore) PlanMigration(ctx context.Context) (*MigrationPlan, error) {
	roots, err := fs.Client.Collection(FirestoreCollection).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("read domain documents: %w", err)
	}
	records := make([]MigrationRecord, 0, len(roots))
	for _, snap := range roots {
		records = append(records, MigrationRecord{Path: "links/" + snap.Ref.ID, Data: snap.Data(), UpdateTime: snap.UpdateTime})
	}
	for _, collection := range []string{"id", PathCollection} {
		snaps, err := fs.Client.CollectionGroup(collection).Documents(ctx).GetAll()
		if err != nil {
			return nil, fmt.Errorf("read %s documents: %w", collection, err)
		}
		for _, snap := range snaps {
			parent := snap.Ref.Parent.Parent
			if parent == nil || parent.Parent == nil || parent.Parent.ID != FirestoreCollection || parent.Parent.Parent != nil {
				continue
			}
			records = append(records, MigrationRecord{Path: strings.Join([]string{FirestoreCollection, parent.ID, collection, snap.Ref.ID}, "/"), Data: snap.Data(), UpdateTime: snap.UpdateTime})
		}
	}
	return planMigration(records)
}

func planMigration(records []MigrationRecord) (*MigrationPlan, error) {
	plan := &MigrationPlan{Records: records, Entries: make([]MigrationEntry, 0)}
	occupied := make(map[string]string)
	entryByTarget := make(map[string]int)
	for _, r := range records {
		parts := strings.Split(r.Path, "/")
		if len(parts) == 4 && parts[2] == PathCollection {
			occupied[r.Path] = r.Path
		}
	}
	for _, r := range records {
		parts := strings.Split(r.Path, "/")
		if len(parts) != 2 && (len(parts) != 4 || parts[2] != "id") {
			continue
		}
		if parts[0] != FirestoreCollection {
			return nil, fmt.Errorf("unexpected source collection: %s", r.Path)
		}
		if len(r.Data) == 0 {
			plan.Placeholders++
			continue
		}
		for k, v := range r.Data {
			if _, ok := v.(string); !ok {
				return nil, fmt.Errorf("%s: non-string field %s cannot be represented by this migration backup", r.Path, k)
			}
		}
		to, ok := r.Data["to"].(string)
		if !ok {
			return nil, fmt.Errorf("%s: missing or non-string destination", r.Path)
		}
		if to == "" {
			plan.EmptyDestinations++
		}
		if owner, _ := r.Data["owner"].(string); owner == "" {
			plan.MissingOwners++
		}
		var from *url.URL
		source, hasSource := r.Data["from"]
		if hasSource && source != "" {
			text, ok := source.(string)
			if !ok {
				return nil, fmt.Errorf("%s: non-string source", r.Path)
			}
			var err error
			from, err = url.Parse(text)
			if err != nil || from.Host != parts[1] {
				return nil, fmt.Errorf("%s: source metadata does not match its domain", r.Path)
			}
			if len(parts) == 4 && strings.ReplaceAll(from.Path, "/", "+") != parts[3] {
				return nil, fmt.Errorf("%s: source metadata does not match its legacy key", r.Path)
			}
			if len(parts) == 2 && from.Path != "" {
				return nil, fmt.Errorf("%s: root source metadata contains a path", r.Path)
			}
		} else {
			plan.InferredPaths++
			p := "/"
			if len(parts) == 4 {
				p = strings.ReplaceAll(parts[3], "+", "/")
				if strings.Count(parts[3], "+") > 1 {
					plan.AmbiguousKeys++
				}
				if !strings.HasPrefix(p, "/") {
					p = "/" + p
					plan.RelativePaths++
				}
			}
			from = &url.URL{Host: parts[1], Path: p}
		}
		from = canonicalSource(from)
		target := urlToPath(from)
		id := target[strings.LastIndex(target, "/")+1:]
		if len(id) > 1500 {
			return nil, fmt.Errorf("%s: encoded path exceeds Firestore's document ID limit", r.Path)
		}
		data := make(map[string]interface{}, len(r.Data)+1)
		for k, v := range r.Data {
			data[k] = v
		}
		data["from"] = from.String()
		if previous, exists := occupied[target]; exists {
			if i, ok := entryByTarget[target]; ok && reflect.DeepEqual(plan.Entries[i].Data, data) {
				plan.Entries[i].Duplicates = append(plan.Entries[i].Duplicates, r)
				plan.IdenticalDuplicates++
				continue
			}
			return nil, fmt.Errorf("target conflict between %s and %s", previous, r.Path)
		}
		occupied[target] = r.Path
		entryByTarget[target] = len(plan.Entries)
		plan.Entries = append(plan.Entries, MigrationEntry{Source: r, Target: target, Data: data})
	}
	sort.Slice(plan.Entries, func(i, j int) bool { return plan.Entries[i].Source.Path < plan.Entries[j].Source.Path })
	return plan, nil
}

// ApplyMigration moves each record atomically. Stop application writers first.
// Source update-time checks prevent applying a stale dry run.
func (fs Firestore) ApplyMigration(ctx context.Context, plan *MigrationPlan) error {
	for _, entry := range plan.Entries {
		err := fs.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, source := range entry.sources() {
				snap, err := tx.Get(fs.Client.Doc(source.Path))
				if err != nil {
					return err
				}
				if !snap.UpdateTime.Equal(source.UpdateTime) {
					return fmt.Errorf("source changed since planning: %s", source.Path)
				}
			}
			if err := tx.Create(fs.Client.Doc(entry.Target), entry.Data); err != nil {
				return err
			}
			for _, source := range entry.sources() {
				if err := tx.Delete(fs.Client.Doc(source.Path)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("migrate %s: %w", entry.Source.Path, err)
		}
	}
	return fs.VerifyMigration(ctx, plan)
}

// VerifyMigration checks every moved document and confirms the old keys are gone.
func (fs Firestore) VerifyMigration(ctx context.Context, plan *MigrationPlan) error {
	for _, entry := range plan.Entries {
		snap, err := fs.Client.Doc(entry.Target).Get(ctx)
		if err != nil {
			return fmt.Errorf("verify target %s: %w", entry.Target, err)
		}
		if !reflect.DeepEqual(snap.Data(), entry.Data) {
			return fmt.Errorf("target contents differ: %s", entry.Target)
		}
		for _, source := range entry.sources() {
			_, err = fs.Client.Doc(source.Path).Get(ctx)
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("legacy source still present or unreadable: %s", source.Path)
			}
		}
	}
	return nil
}

// RollbackMigration restores a saved plan atomically, refusing changed targets.
func (fs Firestore) RollbackMigration(ctx context.Context, plan *MigrationPlan) error {
	for _, entry := range plan.Entries {
		err := fs.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			target := fs.Client.Doc(entry.Target)
			allRestored := true
			anyOccupied := false
			for _, source := range entry.sources() {
				old, err := tx.Get(fs.Client.Doc(source.Path))
				if err != nil && status.Code(err) != codes.NotFound {
					return err
				}
				if err == nil {
					anyOccupied = true
				}
				if err != nil || !reflect.DeepEqual(old.Data(), source.Data) {
					allRestored = false
				}
			}
			current, currentErr := tx.Get(target)
			if status.Code(currentErr) == codes.NotFound && allRestored {
				return nil
			}
			if currentErr != nil {
				return currentErr
			}
			if !reflect.DeepEqual(current.Data(), entry.Data) {
				return fmt.Errorf("target changed since migration: %s", entry.Target)
			}
			if anyOccupied {
				return fmt.Errorf("legacy source occupied: %s", entry.Source.Path)
			}
			for _, source := range entry.sources() {
				if err := tx.Create(fs.Client.Doc(source.Path), source.Data); err != nil {
					return err
				}
			}
			return tx.Delete(target)
		})
		if err != nil {
			return fmt.Errorf("rollback %s: %w", entry.Source.Path, err)
		}
	}
	return nil
}

func (e MigrationEntry) sources() []MigrationRecord {
	return append([]MigrationRecord{e.Source}, e.Duplicates...)
}

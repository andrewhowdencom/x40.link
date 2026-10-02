package firestore

import (
	"strings"
	"testing"
)

func TestPlanMigration(t *testing.T) {
	records := []MigrationRecord{
		{Path: "links/example.com", Data: map[string]interface{}{}},
		{Path: "links/other.com", Data: map[string]interface{}{"to": "https://root.example"}},
		{Path: "links/example.com/id/+foo+bar", Data: map[string]interface{}{"to": "https://one.example", "owner": "owner"}},
		{Path: "links/example.com/id/relative+path", Data: map[string]interface{}{"to": "", "owner": "owner"}},
		{Path: "links/example.com/id/+literal+plus", Data: map[string]interface{}{"to": "https://two.example", "owner": "owner", "from": "//example.com/literal+plus"}},
	}
	plan, err := planMigration(records)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 4 || plan.Placeholders != 1 || plan.InferredPaths != 3 || plan.AmbiguousKeys != 1 || plan.RelativePaths != 1 || plan.EmptyDestinations != 1 || plan.MissingOwners != 1 {
		t.Fatalf("unexpected summary: %+v", plan.Summary())
	}
	wants := map[string]string{"links/other.com": "//other.com/", "links/example.com/id/+foo+bar": "//example.com/foo/bar", "links/example.com/id/relative+path": "//example.com/relative/path", "links/example.com/id/+literal+plus": "//example.com/literal+plus"}
	for _, e := range plan.Entries {
		if got := e.Data["from"]; got != wants[e.Source.Path] {
			t.Errorf("%s from = %v", e.Source.Path, got)
		}
		if !strings.Contains(e.Target, "/shortLinks/p-") {
			t.Errorf("unexpected target %q", e.Target)
		}
		if e.Data["to"] != e.Source.Data["to"] {
			t.Errorf("destination changed")
		}
	}
}

func TestPlanMigrationRejectsConflicts(t *testing.T) {
	tests := []struct {
		name    string
		records []MigrationRecord
	}{
		{"duplicate canonical path", []MigrationRecord{{Path: "links/example.com/id/+foo", Data: map[string]interface{}{"to": "one"}}, {Path: "links/example.com/id/foo", Data: map[string]interface{}{"to": "two"}}}},
		{"root conflict", []MigrationRecord{{Path: "links/example.com", Data: map[string]interface{}{"to": "one"}}, {Path: "links/example.com/id/+", Data: map[string]interface{}{"to": "two"}}}},
		{"invalid source metadata", []MigrationRecord{{Path: "links/example.com/id/+foo", Data: map[string]interface{}{"to": "one", "from": "//elsewhere.com/foo"}}}},
		{"invalid destination type", []MigrationRecord{{Path: "links/example.com/id/+foo", Data: map[string]interface{}{"to": 12}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := planMigration(tt.records); err == nil {
				t.Fatal("expected refusal")
			}
		})
	}
}

func TestPlanMigrationConsolidatesIdenticalRecords(t *testing.T) {
	plan, err := planMigration([]MigrationRecord{
		{Path: "links/example.com/id/+foo", Data: map[string]interface{}{"to": "same", "owner": "owner"}},
		{Path: "links/example.com/id/foo", Data: map[string]interface{}{"to": "same", "owner": "owner"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 1 || len(plan.Entries[0].Duplicates) != 1 || plan.IdenticalDuplicates != 1 {
		t.Fatalf("unexpected consolidation: %+v", plan.Summary())
	}
}

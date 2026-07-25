package migrate

import "testing"

func TestMigrationListIsOrdered(t *testing.T) {
	files, err := migrationList()
	if err != nil {
		t.Fatalf("expected migration list, got %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one migration, got %d", len(files))
	}
	if files[0] != "migrations/0001_extensions.sql" {
		t.Fatalf("unexpected migration file %q", files[0])
	}
}

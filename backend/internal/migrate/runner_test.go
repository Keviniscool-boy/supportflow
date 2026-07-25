package migrate

import "testing"

func TestMigrationListIsOrdered(t *testing.T) {
	files, err := migrationList()
	if err != nil {
		t.Fatalf("expected migration list, got %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two migrations, got %d", len(files))
	}
	if files[0] != "migrations/0001_extensions.sql" || files[1] != "migrations/0002_schema.sql" {
		t.Fatalf("unexpected migration files %q", files)
	}
}

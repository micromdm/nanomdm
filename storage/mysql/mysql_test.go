package mysql

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/micromdm/nanomdm/test/e2e"
)

func clearAllCommands(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, q := range []string{
		"DELETE FROM enrollment_queue;",
		"DELETE FROM command_results;",
		"DELETE FROM commands;",
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMySQL(t *testing.T) {
	testDSN := os.Getenv("NANOMDM_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANOMDM_MYSQL_STORAGE_TEST_DSN not set")
	}

	s, err := New(WithDSN(testDSN), WithDeleteCommands())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	clearAllCommands(t, ctx, s.db)
	t.Run("e2e-WithDeleteCommands()", func(t *testing.T) { e2e.TestE2E(t, ctx, s) })

	s, err = New(WithDSN(testDSN))
	if err != nil {
		t.Fatal(err)
	}

	clearAllCommands(t, ctx, s.db)
	t.Run("e2e", func(t *testing.T) { e2e.TestE2E(t, ctx, s) })

}

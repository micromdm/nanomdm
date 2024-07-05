package mysql

import (
	"context"
	"os"
	"testing"

	"github.com/micromdm/nanomdm/storage/test"
)

func TestRetrievePushInfo(t *testing.T) {
	testDSN := os.Getenv("NANOMDM_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANOMDM_MYSQL_STORAGE_TEST_DSN not set")
	}

	storage, err := New(WithDSN(testDSN), WithDeleteCommands())
	if err != nil {
		t.Fatal(err)
	}

	test.TestRetrievePushInfo(t, context.Background(), storage)
}

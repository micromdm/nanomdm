package diskv

import (
	"context"
	"testing"

	"github.com/micromdm/nanomdm/test/e2e"
)

func TestDiskv(t *testing.T) {
	t.Run("e2e", func(t *testing.T) { e2e.TestE2E(t, context.Background(), New(t.TempDir())) })
}

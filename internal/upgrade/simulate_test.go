package upgrade

import (
	"context"
	"testing"

	"github.com/TEHILLA-O/forkledger/internal/config"
	"github.com/TEHILLA-O/forkledger/internal/logging"
	"github.com/TEHILLA-O/forkledger/internal/testutil"
)

func TestSimulateDoesNotTouchUserHEAD(t *testing.T) {
	_, fork := testutil.BuildForkWorld(t)
	before := testutil.Hash(t, fork, "HEAD")
	res, err := Simulate(context.Background(), fork, "upstream/main", config.Default(), logging.New(logging.LevelQuiet), false)
	if err != nil {
		t.Fatal(err)
	}
	after := testutil.Hash(t, fork, "HEAD")
	if before != after {
		t.Fatal("upgrade simulation must not move HEAD in the operator repository")
	}
	if !res.UserTreeUntouched {
		t.Fatal("expected UserTreeUntouched")
	}
	if res.Evaluated < 1 {
		t.Fatalf("expected to evaluate patches, %+v", res)
	}
}

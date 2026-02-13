package repository

import (
	"os"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.SetupTestMain(m))
}

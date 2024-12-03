package procfs_test

import (
	"strings"
	"testing"

	// "github.com/davecgh/go-spew/spew"
	"github.com/jessegalley/nfsmountstats/internal/procfs"
	"github.com/stretchr/testify/assert"
)

// TestPrefixMountstatsPath sets the PathPrefix to an arbitrary value and then
// checks whether the resulting path is as expected.
func TestPrefixMountstatsPath(t *testing.T) {
  procfs.PathPrefix = "foobar"
  path := procfs.GetMountstatsPath()
  assert.Equal(t, "foobar/proc/self/mountstats", path)
}

// TestReadMountstatsFile reads and example `/proc/self/mountstats` file prefixed  
// with `testdata` in order to read the file in the local package directory.
// Will fail if the file can't be read or if the content doesn't pass basic 
// sniff tests.
func TestReadMountstatsFile(t *testing.T) {
  procfs.PathPrefix = "testdata"
  content, err := procfs.ReadMountstats()
  if err != nil {
    t.Fatalf("failed to read mountstats: %v", err)
  }

  contentStr := string(content)
  // basic smoke tests to make sure the testdata file was read in correctly 
  // it should begin with `device` and have many lines.
  assert.Equal(t, "device", contentStr[:6])
  assert.Less(t, 10, len(strings.Split(contentStr, "\n")))
  // TODO: add more robust tests here, though i'm not yet sure what...
}

package procfs

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
  mountstatsPath = "/proc/self/mountstats" // this path is where mountstats exist on on all linux 
)

var PathPrefix = "" // path to prefix the mountstats constant path with, for testing purposes 

// ReadMountstats opens the `/proc/self/mountstats` and reads it's
// entire contents into a byte slice.
// Returns non-nil error if the file could not be read. Never returns `io.EOF`.
func ReadMountstats() ([]byte, error) {
  path := GetMountstatsPath()
  
  content, err := os.ReadFile(path)
  if err != nil {
    return nil, fmt.Errorf("failed to read mountstats file (%v)", err)
  }

  return content, nil
} 

// GetMountstatsPath returns the mountstats path appended with any 
// package level prefix that was set with PathPrefix
func GetMountstatsPath() (string) {
  return filepath.Join(PathPrefix,mountstatsPath)
}

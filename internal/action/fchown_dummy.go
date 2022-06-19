//go:build !(linux || darwin)

package action

import (
	"os"
)

// tryFChown tries to copy owner and group information form one file to another if any owner information is available. It returns true on success and false if no file owner and group information is available. This function is a dummy, no-op implementation, that always return false and nil error, when the given system is not supported.
func tryFChown(dst *os.File, srcStat os.FileInfo) (bool, error) {
	return false, nil
}

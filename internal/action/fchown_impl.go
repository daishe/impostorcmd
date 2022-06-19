//go:build linux || darwin

package action

import (
	"os"
	"syscall"
)

// tryFChown tries to copy owner and group information form one file to another if any owner information is available. It returns true on success and false if no file owner and group information is available. This function is a dummy, no-op implementation, that always return false and nil error, when the given system is not supported.
func tryFChown(dst *os.File, srcStat os.FileInfo) (bool, error) {
	srcSys := srcStat.Sys()
	if srcSys == nil {
		return false, nil
	}
	srcSysStat, ok := srcSys.(*syscall.Stat_t)
	if !ok {
		return false, nil
	}
	if err := dst.Chown(int(srcSysStat.Uid), int(srcSysStat.Gid)); err != nil {
		return false, err
	}
	return true, nil
}

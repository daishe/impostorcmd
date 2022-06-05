package action

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"

	"google.golang.org/protobuf/proto"

	"github.com/daishe/impostorcmd/internal/descriptor"
	impostordatav1 "github.com/daishe/impostorcmd/internal/impostordata/v1"
)

func Install(target *impostordatav1.TargetDescriptor) (Compensate, error) {
	target = proto.Clone(target).(*impostordatav1.TargetDescriptor)
	c := Compensate(nil)

	selfPath, err := os.Executable()
	if err != nil {
		return c, fmt.Errorf("obtaining impostorcmd: %w", err)
	}

	originalCmd := target.OriginalCmd
	originalCmdMoved, err := appendRandomPathSuffixFileNoExists(target.OriginalCmd)
	if err != nil {
		return c, fmt.Errorf("moving original command: %w", err)
	}
	target.OriginalCmd = originalCmdMoved

	mvUndo, err := mv(originalCmdMoved, originalCmd)
	c.With(mvUndo)
	if err != nil {
		return c, fmt.Errorf("moving original command: %w", err)
	}

	copy := func(dst *os.File, src *os.File) error {
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		if err := descriptor.AppendToExecutable(dst, target); err != nil {
			return err
		}
		return dst.Sync()
	}
	cpUndo, err := cp(originalCmd, selfPath, originalCmdMoved, copy)
	c.With(cpUndo)
	if err != nil {
		return c, fmt.Errorf("attempting to impostor command: %w", err)
	}

	return c, nil
}

func Uninstall(cmd string) (Compensate, error) {
	c := Compensate(nil)

	cmd, err := descriptor.Lookup(cmd)
	if err != nil {
		return c, err
	}

	desc, err := loadDescriptor(cmd)
	if err != nil {
		return c, err
	}

	cmdTmp, err := appendRandomPathSuffixFileNoExists(cmd)
	if err != nil {
		return c, fmt.Errorf("moving impostor command: %w", err)
	}
	mvCmdUndo, err := mv(cmdTmp, cmd)
	c.With(mvCmdUndo)
	if err != nil {
		return c, fmt.Errorf("moving impostor command: %w", err)
	}

	mvOriginalUndo, err := mv(cmd, desc.OriginalCmd)
	c.With(mvOriginalUndo)
	if err != nil {
		return c, fmt.Errorf("moving original command: %w", err)
	}

	if err = os.Remove(cmdTmp); err != nil {
		return c, fmt.Errorf("removing impostor command: %w", err)
	}
	return nil, nil // removal cannot be undone
}

func loadDescriptor(path string) (*impostordatav1.TargetDescriptor, error) {
	cmdFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading command file: %w", err)
	}
	defer cmdFile.Close()
	desc, err := descriptor.FromExecutable(cmdFile)
	if err != nil {
		return nil, fmt.Errorf("while reading %s: %w", path, err)
	}
	return desc, nil
}

func appendRandomPathSuffixFileNoExists(path string) (string, error) {
	for {
		suffixBytes := make([]byte, 16)
		if _, err := rand.Read(suffixBytes); err != nil {
			return "", err
		}
		suffix := "-" + hex.EncodeToString(suffixBytes)

		newPath := ""
		switch {
		case runtime.GOOS == "windows" && strings.HasSuffix(path, ".exe"):
			newPath = strings.TrimSuffix(path, ".exe") + suffix + ".exe"
		case runtime.GOOS == "windows" && strings.HasSuffix(path, ".bat"):
			newPath = strings.TrimSuffix(path, ".bat") + suffix + ".bat"
		default:
			newPath = path + suffix
		}

		if _, err := os.Stat(newPath); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return newPath, nil
	}
}

func mv(dst, src string) (undo Compensate, err error) {
	err = os.Rename(src, dst)
	if err != nil {
		return undo, err
	}
	undo = func() error {
		return os.Rename(dst, src)
	}
	return undo, nil
}

func cp(dst, src string, modOwnerRef string, copy func(*os.File, *os.File) error) (undo Compensate, err error) {
	refStat, err := os.Stat(modOwnerRef)
	if err != nil {
		return undo, err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return undo, err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, refStat.Mode())
	if err != nil {
		return undo, err
	}
	defer dstFile.Close()
	undo = func() error {
		return os.Remove(dst)
	}

	if err = copy(dstFile, srcFile); err != nil {
		return undo, err
	}

	if runtime.GOOS == "windows" {
		return undo, nil
	}
	refSys := refStat.Sys()
	if refSys == nil {
		return undo, nil
	}
	refStatSys, ok := refSys.(*syscall.Stat_t)
	if !ok {
		return undo, nil
	}
	if err = dstFile.Chown(int(refStatSys.Uid), int(refStatSys.Gid)); err != nil {
		return undo, err
	}
	return undo, nil
}

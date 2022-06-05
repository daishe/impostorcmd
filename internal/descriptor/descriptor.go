package descriptor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"google.golang.org/protobuf/proto"

	configv1 "github.com/daishe/impostorcmd/config/v1"
	impostordatav1 "github.com/daishe/impostorcmd/internal/impostordata/v1"
)

const fileMagic = "IMPOSTOR"
const fileMagicBytesLen = len(fileMagic)

const descriptorMaxSize = 10 * 1024 * 1024 // descriptor maximum size - 10 MiB is more than enough
const descriptorSizeBytesLen = 4

var descriptorSizeEncoding binary.ByteOrder = binary.BigEndian

type ErrorNoDescriptor struct {
}

func (e ErrorNoDescriptor) Error() string {
	return "not an impostor (data do not contain an impostor descriptor)"
}

type ErrorDescriptorUnsupportedVersion struct {
	Version string
}

func (e ErrorDescriptorUnsupportedVersion) Error() string {
	if e.Version == "" {
		return "empty impostor descriptor version is unsupported"
	}
	return fmt.Sprintf("impostor descriptor version %s is unsupported", e.Version)
}

func Lookup(path string) (absPath string, err error) {
	clean := func(path string) (string, error) {
		path, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		return filepath.Abs(path)
	}

	if !strings.ContainsRune(path, os.PathSeparator) { // not a path
		if path, err = exec.LookPath(path); err != nil {
			return "", err
		}
	}
	if absPath, err = filepath.Abs(path); err != nil {
		return "", err
	}

	if _, err = os.Stat(absPath); err == nil {
		return clean(absPath)
	}
	if runtime.GOOS == "windows" {
		if _, err = os.Stat(absPath + ".exe"); err == nil {
			return clean(absPath + ".exe")
		}
		if _, err = os.Stat(absPath + ".bat"); err == nil {
			return clean(absPath + ".bat")
		}
	}
	return "", fmt.Errorf("cannot find file under path %s", absPath)
}

func FromTarget(target *configv1.Target) (*impostordatav1.TargetDescriptor, error) {
	cmd, err := Lookup(target.GetCmd())
	if err != nil {
		return nil, fmt.Errorf("cannot find command %s: %w", target.Cmd, err)
	}

	desc := &impostordatav1.TargetDescriptor{
		Version:         "v1",
		OriginalCmd:     cmd,
		ImpostorCmd:     target.GetImpostor(),
		ImpostorCmdArgs: target.GetImpostorArgs(),
		IncludeArg_0:    target.GetIncludeArg_0(),
	}
	return desc, nil
}

func FromExecutable(r io.ReadSeeker) (*impostordatav1.TargetDescriptor, error) {
	if size, err := r.Seek(0, io.SeekEnd); err != nil {
		return nil, fmt.Errorf("reading impostor descriptor: %w", err)
	} else if size < int64(descriptorSizeBytesLen+fileMagicBytesLen) {
		return nil, ErrorNoDescriptor{}
	}

	if _, err := r.Seek(-int64(descriptorSizeBytesLen+fileMagicBytesLen), io.SeekEnd); err != nil {
		return nil, fmt.Errorf("reading impostor descriptor: %w", err)
	}

	sizeAndMagicBytes := [descriptorSizeBytesLen + fileMagicBytesLen]byte{}
	if _, err := io.ReadFull(r, sizeAndMagicBytes[:]); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, ErrorNoDescriptor{}
		}
		return nil, fmt.Errorf("reading impostor descriptor: %w", err)
	}
	sizeBytes, magicBytes := sizeAndMagicBytes[0:descriptorSizeBytesLen], sizeAndMagicBytes[descriptorSizeBytesLen:]
	if !bytes.Equal(magicBytes, []byte("IMPOSTOR")) {
		return nil, ErrorNoDescriptor{}
	}
	size := descriptorSizeEncoding.Uint32(sizeBytes)
	if size > descriptorMaxSize { // 10 MiB is more than enough
		return nil, fmt.Errorf("impostor descriptor is too large")
	}
	if size == 0 {
		return nil, fmt.Errorf("impostor descriptor is empty")
	}

	if _, err := r.Seek(-int64(descriptorSizeBytesLen+fileMagicBytesLen)-int64(size), io.SeekEnd); err != nil {
		return nil, fmt.Errorf("reading impostor descriptor: %w", err)
	}

	descBytes := make([]byte, int(size))
	if _, err := io.ReadFull(r, descBytes); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("reading impostor descriptor: invalid descriptor size")
		}
		return nil, fmt.Errorf("reading impostor descriptor: %w", err)
	}
	descVer := &impostordatav1.ObjectVersion{}
	if err := proto.Unmarshal(descBytes, descVer); err != nil {
		return nil, fmt.Errorf("unmarshalling impostor descriptor version: %w", err)
	}
	if err := checkVersionString(descVer.Version); err != nil {
		return nil, fmt.Errorf("unmarshalling impostor descriptor: %w", err)
	}
	desc := &impostordatav1.TargetDescriptor{}
	if err := proto.Unmarshal(descBytes, desc); err != nil {
		return nil, fmt.Errorf("unmarshalling impostor descriptor: %w", err)
	}
	if err := checkVersionString(desc.Version); err != nil {
		return nil, fmt.Errorf("unmarshalling impostor descriptor: %w", err)
	}
	return desc, nil
}

func checkVersionString(v string) error {
	if v != "v1" {
		return ErrorDescriptorUnsupportedVersion{v}
	}
	return nil
}

func AppendToExecutable(w io.WriteSeeker, desc *impostordatav1.TargetDescriptor) error {
	if err := checkVersionString(desc.Version); err != nil {
		return fmt.Errorf("marshalling impostor descriptor: %w", err)
	}
	descBytes, err := proto.Marshal(desc)
	if err != nil {
		return fmt.Errorf("marshalling impostor descriptor: %w", err)
	}
	if len(descBytes) > descriptorMaxSize {
		return fmt.Errorf("impostor descriptor is too large")
	}
	sizeAndMagicBytes := [descriptorSizeBytesLen + fileMagicBytesLen]byte{}
	descriptorSizeEncoding.PutUint32(sizeAndMagicBytes[0:descriptorSizeBytesLen], uint32(len(descBytes)))
	copy(sizeAndMagicBytes[descriptorSizeBytesLen:], []byte(fileMagic))

	descBytes = append(descBytes, sizeAndMagicBytes[:]...)

	if _, err := w.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("writing impostor descriptor: %w", err)
	}
	if _, err := w.Write(descBytes); err != nil {
		return fmt.Errorf("writing impostor descriptor: %w", err)
	}
	return nil
}

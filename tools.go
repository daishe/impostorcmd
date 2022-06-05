//go:build tools

package impostorcmd

import (
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/golang/protobuf/protoc-gen-go"
)

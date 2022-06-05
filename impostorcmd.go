package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/daishe/impostorcmd/cmd"
	"github.com/daishe/impostorcmd/internal/action"
)

var (
	Version              = "development"
	Commit               = "?"
	ConfigurationVersion = "v1"
)

func main() {
	cmd.SetApplicationVersion(Version)
	cmd.SetCommitHash(Commit)
	cmd.SetConfigVersion(ConfigurationVersion)

	isImpostor, desc, err := action.IsCurrentProcessImpostor() // am I an impostor ?
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if isImpostor {
		err := action.Impostor(context.Background(), desc, os.Args...)
		if ee := (&exec.ExitError{}); errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := cmd.Execute(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

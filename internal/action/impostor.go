package action

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"

	"github.com/daishe/impostorcmd/internal/descriptor"
	impostordatav1 "github.com/daishe/impostorcmd/internal/impostordata/v1"
)

func IsCurrentProcessImpostor() (bool, *impostordatav1.TargetDescriptor, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return false, nil, fmt.Errorf("obtaining path to current process executable: %w", err)
	}
	self, err := os.Open(selfPath)
	if err != nil {
		return false, nil, fmt.Errorf("reading current process executable: %w", err)
	}
	defer self.Close()

	desc, err := descriptor.FromExecutable(self)
	if err != nil {
		if !errors.As(err, &descriptor.ErrorNoDescriptor{}) {
			return false, nil, fmt.Errorf("reading self target descriptor: %w", err)
		}
		return false, nil, nil
	}
	return true, desc, nil
}

func Impostor(ctx context.Context, target *impostordatav1.TargetDescriptor, args ...string) error {
	cmdArgs := make([]string, 0, len(target.GetImpostorCmdArgs())+len(args))
	cmdArgs = append(cmdArgs, target.GetImpostorCmdArgs()...)
	if target.IncludeArg_0 {
		cmdArgs = append(cmdArgs, args...)
	} else {
		cmdArgs = append(cmdArgs, args[1:]...)
	}

	impostorCmdPath, err := descriptor.Lookup(target.ImpostorCmd)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, impostorCmdPath, cmdArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append([]string(nil), os.Environ()...)
	cmd.Env = append(cmd.Env, "IMPOSTORCMD_ORIGINAL_COMMAND="+target.OriginalCmd)
	if err := cmd.Start(); err != nil {
		return err
	}

	sigpassStop := sigpass(ctx, cmd)
	err = cmd.Wait()
	sigpassStop()
	return err
}

func sigpass(ctx context.Context, cmd *exec.Cmd) func() {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, ctxCancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}
	ch := make(chan os.Signal, 1024)
	routine := func() {
		signal.Notify(ch)
		for {
			select {
			case sig := <-ch:
				// ignore errors
				cmd.Process.Signal(sig) //nolint:errcheck
			case <-ctx.Done():
				signal.Stop(ch)
				wg.Done()
				return
			}
		}
	}
	stop := func() {
		ctxCancel()
		wg.Wait()
	}
	wg.Add(1)
	go routine()
	return stop
}

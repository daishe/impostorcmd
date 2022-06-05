package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	configv1 "github.com/daishe/impostorcmd/config/v1"
	"github.com/daishe/impostorcmd/internal/action"
	"github.com/daishe/impostorcmd/internal/config"
	"github.com/daishe/impostorcmd/internal/descriptor"
	impostordatav1 "github.com/daishe/impostorcmd/internal/impostordata/v1"
)

type installOptions struct {
	json        string
	config      string
	includeArg0 bool
}

func installCmd(r *rootOptions) *cobra.Command {
	o := &installOptions{}
	cmd := &cobra.Command{
		Use:   "install [option]... target-command impostor-command [argument]...",
		Short: "setup impostoring scheme",
		Long:  "Start impostoring command or commands.",
	}
	cmd.Flags().StringVar(&o.json, "json", "", "JSON setup description for single target")
	cmd.Flags().StringVar(&o.config, "config", "", "JSON configuration file containing setup description")
	cmd.Flags().BoolVar(&o.includeArg0, "include-arg-0", false, "include argument #0 from original command when invoking impostor command")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		checkErr(cmd, installCmdRun(cmd, r, o, args))
	}
	return cmd
}

func installCmdRun(cmd *cobra.Command, r *rootOptions, o *installOptions, args []string) (err error) {
	isByInlineJson, isByConfig, isByArgs := o.json != "", o.config != "", len(args) > 0
	trueCount := func(x ...bool) (count int) {
		for _, v := range x {
			if v {
				count++
			}
		}
		return count
	}

	switch {
	case trueCount(isByArgs, isByInlineJson, isByConfig) == 0:
		return fmt.Errorf("no arguments, 'json' flag nor 'config' flag specified")
	case trueCount(isByArgs, isByInlineJson, isByConfig) == 2:
		l := make([]string, 0, 2)
		if isByArgs {
			l = append(l, "arguments")
		}
		if isByInlineJson {
			l = append(l, "'json' flag")
		}
		if isByConfig {
			l = append(l, "'config' flag")
		}
		return fmt.Errorf("%s specified together", strings.Join(l, " and "))
	case trueCount(isByArgs, isByInlineJson, isByConfig) == 3:
		return fmt.Errorf("arguments, 'json' flag and 'config' flag specified together")
	}

	targetDescs := []*impostordatav1.TargetDescriptor(nil)
	switch {
	case isByInlineJson:
		targetDescs, err = targetDescriptorByJsonTarget(cmd.Context(), o.json)
	case isByConfig:
		targetDescs, err = targetDescriptorByConfigFile(cmd.Context(), o.config)
	default: // isByArgs
		targetDescs, err = targetDescriptorByInstallArgs(cmd.Context(), r, o, args)
	}
	if err != nil {
		return err
	}

	undoAll := action.Compensate(nil)
	wrapUndo := func(undo action.Compensate, cmd *cobra.Command, target *impostordatav1.TargetDescriptor) func() error {
		return func() error {
			if err := undo(); err != nil {
				showErr(cmd, fmt.Errorf("undoing actions taken for target %s failed: %w", target.OriginalCmd, err))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Undone actions taken for target %s\n", target.OriginalCmd)
			}
			return nil
		}
	}

	for _, t := range targetDescs {
		undo, err := action.Install(t)
		undoAll.With(wrapUndo(undo, cmd, t))
		if err != nil {
			showErr(cmd, fmt.Errorf("installing target %s failed: %w", t.OriginalCmd, err))
			showErr(cmd, undoAll.Run())
			return fmt.Errorf("failure occurred while attempting to impostor target %s", t.OriginalCmd)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Installed impostor for target %s\n", t.OriginalCmd)
	}
	return nil
}

func targetDescriptorByJsonTarget(ctx context.Context, json string) ([]*impostordatav1.TargetDescriptor, error) {
	target, err := config.UnmarshalAndValidateTarget([]byte(json))
	if err != nil {
		return nil, fmt.Errorf("parsing 'json' flag value: %w", err)
	}
	desc, err := descriptor.FromTarget(target)
	if err != nil {
		return nil, err
	}
	return []*impostordatav1.TargetDescriptor{desc}, nil
}

func targetDescriptorByConfigFile(ctx context.Context, configPath string) ([]*impostordatav1.TargetDescriptor, error) {
	cfgBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading configuration file: %w", err)
	}
	cfg, err := config.UnmarshalAndValidateConfiguration(cfgBytes)
	if err != nil {
		return nil, err
	}
	descs := make([]*impostordatav1.TargetDescriptor, 0, len(cfg.Targets))
	for i, t := range cfg.Targets {
		desc, err := descriptor.FromTarget(t)
		if err != nil {
			return nil, fmt.Errorf("target #%d (%s): %w", i+1, t.Cmd, err)
		}
		descs = append(descs, desc)
	}
	return descs, nil
}

func targetDescriptorByInstallArgs(ctx context.Context, r *rootOptions, o *installOptions, args []string) ([]*impostordatav1.TargetDescriptor, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("too few arguments provided: missing target-command and impostor-command")
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("too few arguments provided: missing impostor-command")
	}
	target := &configv1.Target{
		Cmd:          args[0],
		Impostor:     args[1],
		ImpostorArgs: args[2:],
		IncludeArg_0: o.includeArg0,
	}
	desc, err := descriptor.FromTarget(target)
	if err != nil {
		return nil, err
	}
	return []*impostordatav1.TargetDescriptor{desc}, nil
}

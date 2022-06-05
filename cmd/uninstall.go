package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	configv1 "github.com/daishe/impostorcmd/config/v1"
	"github.com/daishe/impostorcmd/internal/action"
	"github.com/daishe/impostorcmd/internal/descriptor"
	impostordatav1 "github.com/daishe/impostorcmd/internal/impostordata/v1"
)

type uninstallOptions struct {
	json   string
	config string
}

func uninstallCmd(r *rootOptions) *cobra.Command {
	o := &uninstallOptions{}
	cmd := &cobra.Command{
		Use:   "uninstall [option]... target-command",
		Short: "undo impostoring scheme",
		Long:  "Stop impostoring command or commands.",
	}
	cmd.Flags().StringVar(&o.json, "json", "", "JSON setup description for single target")
	cmd.Flags().StringVar(&o.config, "config", "", "JSON configuration file containing setup description")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		checkErr(cmd, uninstallCmdRun(cmd, r, o, args))
	}
	return cmd
}

func uninstallCmdRun(cmd *cobra.Command, r *rootOptions, o *uninstallOptions, args []string) (err error) {
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
		targetDescs, err = targetDescriptorByUninstallArgs(cmd.Context(), r, o, args)
	}
	if err != nil {
		return err
	}

	for _, t := range targetDescs {
		undo, err := action.Uninstall(t.OriginalCmd)
		if err != nil {
			if errors.As(err, &descriptor.ErrorNoDescriptor{}) {
				fmt.Fprintf(cmd.OutOrStdout(), "Skipping non impostor target %s\n", t.OriginalCmd)
				continue
			}
			showErr(cmd, fmt.Errorf("uninstalling target %s failed: %w", t.OriginalCmd, err))
			if err := undo.Run(); err != nil {
				showErr(cmd, fmt.Errorf("undoing actions taken for target %s failed: %w", t.OriginalCmd, err))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Undone actions taken for target %s\n", t.OriginalCmd)
			}
			return fmt.Errorf("failure occurred while attempting to uninstall impostor in target %s", t.OriginalCmd)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Uninstalled impostor for target %s\n", t.OriginalCmd)
	}
	return nil
}

func targetDescriptorByUninstallArgs(ctx context.Context, r *rootOptions, o *uninstallOptions, args []string) ([]*impostordatav1.TargetDescriptor, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("too few arguments provided: missing target-command")
	}
	target := &configv1.Target{Cmd: args[0]}
	desc, err := descriptor.FromTarget(target)
	if err != nil {
		return nil, err
	}
	return []*impostordatav1.TargetDescriptor{desc}, nil
}

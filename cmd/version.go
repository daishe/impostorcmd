package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion    string = "development"
	commitHash    string = "?"
	configVersion string = "?"
)

func SetApplicationVersion(app string) {
	appVersion = app
}

func SetCommitHash(commit string) {
	commitHash = commit
}

func SetConfigVersion(config string) {
	configVersion = config
}

type versionOptions struct {
}

func versionCmd(r *rootOptions) *cobra.Command {
	o := &versionOptions{}
	cmd := &cobra.Command{
		Use:   "version",
		Short: "show version information",
		Long:  "Show version information.",
	}
	cmd.Run = func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(versionCmdRun(cmd, r, o, args))
	}
	return cmd
}

func versionCmdRun(cmd *cobra.Command, r *rootOptions, o *versionOptions, args []string) error {
	fmt.Println("application: ", appVersion, ", commit: ", commitHash, ", configuration: ", configVersion)
	return nil
}

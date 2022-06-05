package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type rootOptions struct {
}

func rootCmd() *cobra.Command {
	o := &rootOptions{}
	cmd := &cobra.Command{
		Use:   "impostorcmd",
		Short: "impostor any command",
		Long:  "Impostorcmd allows impostoring any command.",
	}
	cmd.AddCommand(installCmd(o))
	cmd.AddCommand(uninstallCmd(o))
	cmd.AddCommand(versionCmd(o))
	return cmd
}

func showErr(cmd *cobra.Command, msg interface{}) {
	if msg != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", msg)
	}
}

func checkErr(cmd *cobra.Command, msg interface{}) {
	if msg != nil {
		showErr(cmd, msg)
		os.Exit(1)
	}
}

func Execute(ctx context.Context) error {
	return rootCmd().ExecuteContext(ctx)
}

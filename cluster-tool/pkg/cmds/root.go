package cmds

import (
	"flag"

	"github.com/spf13/cobra"
)

var (
	Sanitize bool
)

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "cluster-tool",
		Short:             "cluster-tool by AppsCode - Backup cluster-tool yaml",
		Long:              "cluster-tool is a tool to take restic cluster-tool's yaml using restic",
		DisableAutoGenTag: true,
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	flag.CommandLine.Parse([]string{})

	rootCmd.AddCommand(NewCmdBackup())
	return rootCmd
}

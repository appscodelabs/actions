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
		Short:             "cluster-tool by AppsCode - Backup cluster yaml",
		Long:              "cluster-tool is a tool to take restic cluster's yaml using restic",
		Example:           "cluster-tool restic --sanitize=true --restic-dir=/tmp/restic",
		DisableAutoGenTag: true,
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	flag.CommandLine.Parse([]string{})

	rootCmd.AddCommand(NewCmdBackup())
	return rootCmd
}

package cli

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "shiphold",
		Short: "Git-centric deployment safety and provenance engine",
	}

	rootCmd.AddCommand(newCheckCommand())

	return rootCmd
}

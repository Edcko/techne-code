package cli

import (
	"github.com/spf13/cobra"
)

// newVersionCmd creates the version command.
// It prints version information for the techne binary.
func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Display the version of Techne Code.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("techne %s\n", version)
		},
	}
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				fmt.Println(Version)
				return
			}
			fmt.Printf("mcp-adapter %s\n", Version)
			fmt.Printf("  commit:  %s\n", Commit)
			fmt.Printf("  built:   %s\n", BuildDate)
		},
	}

	cmd.Flags().BoolVarP(&short, "short", "s", false, "Print only the version number")

	return cmd
}

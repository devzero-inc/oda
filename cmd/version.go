package cmd

import (
	"fmt"

	"github.com/devzero-inc/oda/config"

	"github.com/spf13/cobra"
)

// newVersionCmd creates a new version command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Version number of ODA.",
		Long:  `Display ODA version number.`,

		Run: version,
	}
}

func version(_ *cobra.Command, _ []string) {
	fmt.Fprintf(config.SysConfig.Out, "ODA v%s\n", config.Version)
}

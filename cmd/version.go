package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"jutkey-server/packages/consts"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(consts.Version())
	},
}

package cmd

import (
	"github.com/m-ret/armory/internal/tui"
	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Dashboard showing all equipped directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunBoard()
	},
}

func init() {
	rootCmd.AddCommand(boardCmd)
}

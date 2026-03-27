package cmd

import (
	"fmt"
	"os"

	"github.com/m-ret/armory/internal/config"
	"github.com/m-ret/armory/internal/tui"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "armory",
	Short: "Skill control plane for AI agents",
	Long: `armory equips your terminals with team-based skill presets.

Instead of manually symlinking skills for each Claude Code session,
define teams (marketing, dev, planning) and equip them in one command.

  armory scan              List available skills
  armory team create dev   Create a team with interactive picker
  armory equip dev         Equip current directory with the dev team
  armory board             See all equipped directories at a glance`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tui.SetShellVersion(Version)
		firstRun := !config.ConfigExists()
		return tui.RunShell(firstRun)
	},
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version
}

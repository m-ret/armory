package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "armory",
	Short: "Skill control plane for AI agents",
	Long: `armory equips your terminals with role-based skill presets.

Instead of manually symlinking skills for each Claude Code session,
define roles (marketing, dev, planning) and equip them in one command.

  armory scan              List available skills
  armory role create dev   Create a role with interactive picker
  armory equip dev         Equip current directory with the dev role
  armory board             See all equipped directories at a glance`,
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

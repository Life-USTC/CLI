package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/config"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <command>",
		Short: "Manage CLI configuration",
	}
	cmd.AddCommand(newCmdSetServer())
	cmd.AddCommand(newCmdGetServer())
	return cmd
}

func newCmdSetServer() *cobra.Command {
	return &cobra.Command{
		Use:   "set-server <url>",
		Short: "Set the default server URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetDefaultServer(args[0]); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Default server set to %s", args[0]))
			return nil
		},
	}
}

func newCmdGetServer() *cobra.Command {
	return &cobra.Command{
		Use:   "get-server",
		Short: "Show the default server URL",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.GetDefaultServer())
		},
	}
}

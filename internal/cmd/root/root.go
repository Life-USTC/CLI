package root

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/cmd/admin"
	"github.com/Life-USTC/CLI/internal/cmd/authcmd"
	"github.com/Life-USTC/CLI/internal/cmd/bus"
	"github.com/Life-USTC/CLI/internal/cmd/configcmd"
	"github.com/Life-USTC/CLI/internal/cmd/course"
	"github.com/Life-USTC/CLI/internal/cmd/me"
	"github.com/Life-USTC/CLI/internal/cmd/metadata"
	"github.com/Life-USTC/CLI/internal/cmd/schedule"
	"github.com/Life-USTC/CLI/internal/cmd/section"
	"github.com/Life-USTC/CLI/internal/cmd/semester"
	"github.com/Life-USTC/CLI/internal/cmd/teacher"
	"github.com/Life-USTC/CLI/internal/config"
	"github.com/Life-USTC/CLI/internal/output"
)

var version = "dev"

func NewCmdRoot() *cobra.Command {
	var (
		server  string
		format  string
		noColor bool
	)

	cmd := &cobra.Command{
		Use:   "life-ustc <command> <subcommand> [flags]",
		Short: "Life@USTC — command-line client for the USTC campus platform",
		Long:  "Work seamlessly with the USTC campus platform from the command line.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if server == "" {
				server = config.GetDefaultServer()
			}
			output.Current.Format = format
			output.Current.NoColor = noColor
			if noColor {
				color.NoColor = true
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	cmd.SetVersionTemplate("life-ustc version {{.Version}}\n")

	cmd.PersistentFlags().StringVar(&server, "server", "", "Server URL (default: life-ustc.tiankaima.dev, env: LIFE_USTC_SERVER)")
	cmd.PersistentFlags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Register all commands
	cmd.AddCommand(authcmd.NewCmdAuth())
	cmd.AddCommand(me.NewCmdMe())
	cmd.AddCommand(semester.NewCmdSemester())
	cmd.AddCommand(course.NewCmdCourse())
	cmd.AddCommand(section.NewCmdSection())
	cmd.AddCommand(teacher.NewCmdTeacher())
	cmd.AddCommand(schedule.NewCmdSchedule())
	cmd.AddCommand(bus.NewCmdBus())
	cmd.AddCommand(metadata.NewCmdMetadata())
	cmd.AddCommand(admin.NewCmdAdmin())
	cmd.AddCommand(configcmd.NewCmdConfig())
	cmd.AddCommand(newCmdCompletion())

	return cmd
}

func newCmdCompletion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for life-ustc.

To load completions:

  bash:
    source <(life-ustc completion bash)

  zsh:
    life-ustc completion zsh > "${fpath[1]}/_life-ustc"

  fish:
    life-ustc completion fish | source

  powershell:
    life-ustc completion powershell | Out-String | Invoke-Expression`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s (use bash, zsh, fish, or powershell)", args[0])
			}
		},
	}
	return cmd
}

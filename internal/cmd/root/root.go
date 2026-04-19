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

// Command group IDs
const (
	groupCore    = "core"
	groupPersonal = "personal"
	groupBrowse  = "browse"
	groupRef     = "reference"
	groupAdmin   = "admin"
)

func NewCmdRoot() *cobra.Command {
	var (
		server  string
		format  string
		noColor bool
		jq      string
		verbose bool
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "life-ustc <command> <subcommand> [flags]",
		Short: "Life@USTC — command-line client for the USTC campus platform",
		Long: `Work seamlessly with the USTC campus platform from the command line.

Browse courses, sections, and teachers. Manage your todos, homework,
calendar, and uploads. All output supports --jq for scripting.`,
		Example: `  # Show your profile
  life-ustc me

  # List sections and filter with jq
  life-ustc section list --limit 5 --jq '.[].code'

  # Check your pending todos
  life-ustc me todo list --pending

  # View a course and its sections
  life-ustc course view <course-id>

  # Generate shell completions
  life-ustc completion bash`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if server == "" {
				server = config.GetDefaultServer()
			}
			if jsonOut {
				format = "json"
			}
			output.Current.Format = format
			output.Current.NoColor = noColor
			output.Current.JQ = jq
			output.Current.Verbose = verbose
			if noColor {
				color.NoColor = true
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	cmd.SetVersionTemplate("life-ustc version {{.Version}}\n")

	// Global flags
	cmd.PersistentFlags().StringVar(&server, "server", "", "Server URL (default: life-ustc.tiankaima.dev, env: LIFE_USTC_SERVER)")
	cmd.PersistentFlags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().StringVar(&jq, "jq", "", "Filter JSON output with a jq expression")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show verbose output (API requests, timing)")
	cmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output as JSON (shorthand for --format json)")

	// Command groups
	cmd.AddGroup(
		&cobra.Group{ID: groupCore, Title: "Core commands:"},
		&cobra.Group{ID: groupPersonal, Title: "Personal:"},
		&cobra.Group{ID: groupBrowse, Title: "Browse:"},
		&cobra.Group{ID: groupRef, Title: "Reference:"},
		&cobra.Group{ID: groupAdmin, Title: "Administration:"},
	)

	// Core
	authCmd := authcmd.NewCmdAuth()
	authCmd.GroupID = groupCore
	configCmd := configcmd.NewCmdConfig()
	configCmd.GroupID = groupCore
	completionCmd := newCmdCompletion()
	completionCmd.GroupID = groupCore

	// Personal
	meCmd := me.NewCmdMe()
	meCmd.GroupID = groupPersonal

	// Browse
	courseCmd := course.NewCmdCourse()
	courseCmd.GroupID = groupBrowse
	sectionCmd := section.NewCmdSection()
	sectionCmd.GroupID = groupBrowse
	teacherCmd := teacher.NewCmdTeacher()
	teacherCmd.GroupID = groupBrowse
	semesterCmd := semester.NewCmdSemester()
	semesterCmd.GroupID = groupBrowse
	scheduleCmd := schedule.NewCmdSchedule()
	scheduleCmd.GroupID = groupBrowse
	busCmd := bus.NewCmdBus()
	busCmd.GroupID = groupBrowse

	// Reference
	metadataCmd := metadata.NewCmdMetadata()
	metadataCmd.GroupID = groupRef

	// Admin
	adminCmd := admin.NewCmdAdmin()
	adminCmd.GroupID = groupAdmin

	cmd.AddCommand(authCmd, configCmd, completionCmd)
	cmd.AddCommand(meCmd)
	cmd.AddCommand(courseCmd, sectionCmd, teacherCmd, semesterCmd, scheduleCmd, busCmd)
	cmd.AddCommand(metadataCmd)
	cmd.AddCommand(adminCmd)

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

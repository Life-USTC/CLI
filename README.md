# Life@USTC CLI

Command-line client for the [Life@USTC](https://life-ustc.tiankaima.dev) campus platform.

Built with Go, inspired by [GitHub CLI](https://github.com/cli/cli).

## Installation

### From release binaries

Download the latest release from [GitHub Releases](https://github.com/Life-USTC/CLI/releases).

### From source

```bash
go install github.com/Life-USTC/CLI/cmd/life-ustc@latest
```

### Build from source

```bash
git clone https://github.com/Life-USTC/CLI.git
cd CLI
make build
```

## Usage

```bash
# Set server (default: https://life-ustc.tiankaima.dev)
life-ustc --server http://localhost:3000 auth login

# Or set default server
life-ustc config set-server http://localhost:3000

# Authenticate
life-ustc auth login
life-ustc auth status
life-ustc me

# Browse (no auth required)
life-ustc course list --search "数学分析"
life-ustc course view <JW_ID>
life-ustc section list --semester-id <ID>
life-ustc teacher list --search "张"
life-ustc semester list
life-ustc semester current
life-ustc bus query --from east --to west
life-ustc metadata

# Personal features (auth required)
life-ustc todo list
life-ustc todo create --title "Write report" --priority high
life-ustc todo update <ID> --completed
life-ustc todo delete <ID>

life-ustc homework list --section-id <ID>
life-ustc homework complete <ID>

life-ustc comment list --target-type section --target-id <ID>
life-ustc comment create --target-type section --target-id <ID> --body "Great class!"

life-ustc upload list
life-ustc upload file ./report.pdf
life-ustc upload download <ID> -o report.pdf

life-ustc calendar get
life-ustc calendar set <SECTION_ID_1> <SECTION_ID_2>

life-ustc description get --target-type course --target-id <ID>

# Admin
life-ustc admin user list
life-ustc admin comment list --status active
life-ustc admin suspension create --user-id <ID> --reason "spam"
```

## JSON output

All commands support `--format json` for machine-readable output:

```bash
life-ustc --format json semester list
life-ustc --format json course view 12345
```

## Configuration

- Config directory: `~/.config/life-ustc/` (or `$XDG_CONFIG_HOME/life-ustc/`)
- Override server per-command: `--server URL`
- Environment variable: `LIFE_USTC_SERVER`

## Global Options

| Option       | Description                    |
|-------------|--------------------------------|
| `--server`  | Server URL                     |
| `--format`  | Output format (table/json)     |
| `--no-color`| Disable colored output         |
| `--version` | Show version                   |
| `--help`    | Show help                      |

## License

MIT

# Life@USTC CLI

Command-line client for the [Life@USTC](https://life-ustc.tiankaima.dev) server
(Web / REST / GraphQL / MCP live in [Life-USTC/server](https://github.com/Life-USTC/server)).
Domains follow the
[interface hierarchy](https://github.com/Life-USTC/server/blob/main/docs/interface-hierarchy.md).

## Install

```bash
# Release binary: https://github.com/Life-USTC/CLI/releases
go install github.com/Life-USTC/CLI/cmd/life-ustc@latest
# or: git clone … && make build
```

## Usage

```bash
life-ustc --server http://localhost:3000 account login   # or LIFE_USTC_SERVER / config set-server
life-ustc account session|profile|locale zh-cn

life-ustc workspace overview|todo|homework|schedule|exam|subscription|calendar|upload …
life-ustc catalog course|section|teacher|semester|bus|link|metadata …
life-ustc community comment|description|section-homework …
life-ustc admin user|comment|suspension …
life-ustc api catalog/semesters/current                  # raw REST
```

Interactive terminals open a TUI for bare course/section/teacher lists; use
`--no-interactive` for tables. Full flags: `life-ustc <cmd> --help`.

### Domains

| Command | Owns |
|---------|------|
| `catalog` | Public campus facts |
| `workspace` | Current user's work + official-school import |
| `community` | Shared comments / descriptions / section homework |
| `account` | Profile, session, locale |
| `admin` | Governance |
| `config` / `completion` / `api` | CLI plumbing |

### Official USTC sources

`workspace school {semesters,curriculum,exam,score,homework,sync}` talk to
official USTC sites from Go. Flags: `--username`, `--password`, `--totp`,
`--undergraduate` / `--graduate`, `--all-programs`. Defaults via
`life-ustc config set-school-programs` and env
`PASSPORT_{UNDERGRADUATE,GRADUATE}_USERNAME`, `PASSPORT_PASSWORD`, `PASSPORT_TOTP`.

## Output & config

- `--format json` / `--json`, `--jq '…'`
- Config: `~/.config/life-ustc/` (or `$XDG_CONFIG_HOME/life-ustc/`)
- Completion: `life-ustc completion install [--shell zsh|bash]`

| Option | Description |
|--------|-------------|
| `--server` | Server URL |
| `--format` | table / json |
| `--jq` | Filter JSON |
| `--no-color` / `--version` / `--help` | |

## License

MIT

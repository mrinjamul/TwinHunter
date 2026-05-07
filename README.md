# *TwinHunter* (formerly known as `go-dupfinder`)

Production-grade CLI tool to find and clean duplicate files. Written in Go.

## Features

- **Fast scanning** — optimized `WalkDir` with parallel hashing workers
- **Two-stage hashing** — Blake3 for initial grouping, SHA256 for verification
- **Flexible filtering** — min/max size, glob/regex exclusions, directory exclusions
- **Smart cleanup** — delete, replace with hard/soft links, or backup
- **Multiple outputs** — pretty terminal, JSON, CSV, HTML reports
- **Sortable results** — sort duplicate groups by size, count, or path
- **Keep strategies** — auto-select which copy to keep (oldest, newest, shortest path)
- **Dry run** — preview changes before committing to destructive actions
- **Progress indicator** — live file count during scanning
- **Hard link detection** — identifies files that are already hard-linked
- **Shell completion** — bash, zsh, fish, powershell support

## Quick Start

```sh
# Scan current directory
twinhunter

# Scan a specific path (non-recursive)
twinhunter /path/to/dir

# Recursive scan with verbose output
twinhunter find /path/to/dir -r -v

# Export report to JSON
twinhunter find /path/to/dir -r -o report.json

# Delete duplicates (with confirmation prompt)
twinhunter find /path/to/dir -r -d

# Replace with hard links
twinhunter find /path/to/dir -r -l hard

# Dry run preview
twinhunter find /path/to/dir -r -d -n

# Skip confirmation
twinhunter find /path/to/dir -r -d -y
```

## Commands

### `twinhunter [path]`

Shorthand for `twinhunter find [path]`. Scans for duplicate files non-recursively; see `find` below for all options.

### `twinhunter find [path] [flags]`

Full-featured duplicate scan with all options.

| Flag | Short | Description |
|---|---|---|
| `--recursive` | `-r` | Scan subdirectories recursively |
| `--verbose` | `-v` | Show detailed per-group output (hash, size, date) |
| `--min-size` | `-m` | Minimum file size (e.g. `1M`, `100K`, `1G`) |
| `--max-size` | `-M` | Maximum file size (e.g. `10M`) |
| `--exclude` | `-x` | Comma-separated glob patterns to exclude files |
| `--exclude-regex` | | Comma-separated regex patterns to exclude |
| `--exclude-dir` | | Directory names to skip (built-in defaults: `.git,node_modules,.svn,__pycache__,.DS_Store,Thumbs.db`; overrides defaults if set) |
| `--workers` | `-w` | Parallel hashing workers (0 = auto-detect CPU cores) |
| `--output` | `-o` | Export report to file (format auto-detected from extension) |
| `--format` | `-f` | Terminal output: `pretty`, `json`, `csv`, `silent` (default: `pretty`) |
| `--sort` | | Sort groups by: `size`, `count`, `path` (default: `size`) |
| `--keep` | `-k` | Auto-keep strategy: `oldest` (default), `newest`, `shortest` |
| `--link` | `-l` | Replace duplicates with links: `hard`, `soft` |
| `--delete` | `-d` | Delete duplicate files |
| `--yes` | `-y` | Skip confirmation prompts |
| `--dry-run` | `-n` | Preview without making changes |

### `twinhunter clean report.json`

Apply delete/link/backup actions to duplicates listed in a previously exported JSON report.

One action flag is **required** (`--delete`, `--link`, or `--backup-dir`).

| Flag | Short | Description |
|---|---|---|
| `--delete` | `-d` | Delete duplicate files |
| `--link` | `-l` | Replace with links: `hard`, `soft` |
| `--backup-dir` | | Move duplicates to backup directory |
| `--keep` | `-k` | Keep strategy: `oldest` (default), `newest`, `shortest` |
| `--dry-run` | `-n` | Preview without making changes |

### `twinhunter completion [bash|zsh|fish|powershell]`

Generate shell completion script.

```sh
# Bash (current session)
source <(twinhunter completion bash)

# Bash (persistent)
twinhunter completion bash > /etc/bash_completion.d/twinhunter

# Zsh (current session)
source <(twinhunter completion zsh)

# Zsh (persistent)
twinhunter completion zsh > "${fpath[1]}/_twinhunter"

# Fish
twinhunter completion fish > ~/.config/fish/completions/twinhunter.fish

# PowerShell
twinhunter completion powershell | Out-String | Invoke-Expression
```

### `twinhunter version`

Show version and build information.

## Output Formats

### Pretty (terminal)

```
Duplicate Groups: 1
Duplicate Files:  2
Wasted Space:     190 B
Recoverable:      42.2%

Group 1 — 3 copies, 95 B each
  [  keep] /path/to/original.jpg
  [  dup] /path/to/copy1.jpg
  [  dup] /path/to/copy2.jpg
```

Use `-v` for verbose mode with hash, size, and modification time per file.

### JSON

```sh
twinhunter find /path -r -f json
```

Machine-readable output, suitable for piping to `jq` or other tools.

### CSV

```sh
twinhunter find /path -r -f csv
```

Tabular output: `group, hash, size, path, is_duplicate, mod_time`

### Export to File

```sh
# Auto-detects format from extension
twinhunter find /path -r -o report.json
twinhunter find /path -r -o report.csv
twinhunter find /path -r -o report.html
```

The `-o` flag exports to file independently of `-f`. Use both to display CSV/JSON in terminal while saving a different format to file.

## Keep Strategies

When deleting or linking duplicates, TwinHunter needs to know which file to preserve:

| Strategy | Keeps |
|---|---|
| `oldest` | File with earliest modification time (default) |
| `newest` | File with latest modification time |
| `shortest` | File with the shortest file path |

## Interactive Action Prompt

After showing results, TwinHunter presents a numbered menu:

```
Select action:
  1) Delete duplicates
  2) Replace with hard links
  3) Replace with soft links
  4) Skip

Choice [1-4, Enter=4]:
```

- **Enter (no input)** at action prompt = **Skip** (exit cleanly)
- After selecting an action, choose a keep strategy:

```
Keep strategy:
  1) Oldest
  2) Newest
  3) Shortest path

Choice [1-3, Enter=1]:
```

- **Enter** at strategy prompt = **Oldest** (default)

Finally, confirm:

```
Proceed? [y/N]:
```

- **Enter** = No (exit cleanly)
- Type `y` or `yes` to proceed

Use `-y` to skip all prompts when using `--delete` or `--link` flags directly.

Interactive menus only appear in `pretty` terminal output. Formats like `json`, `csv`, and `silent` skip all prompts and exit immediately after reporting results.

## Examples

### Find large duplicates

```sh
twinhunter find /photos -r -m 1M -v
```

### Find and delete, keeping oldest copy

```sh
twinhunter find /documents -r -d -k oldest
```

### Find and keep newest, using shorthand

```sh
twinhunter find /downloads -r -l hard -k newest
```

### Sort by number of copies or path

```sh
twinhunter find /media -r --sort count
twinhunter find /media -r --sort path
```

### Scriptable: silent scan, export to JSON

```sh
twinhunter find /data -r -f silent -o dupes.json
```

### Validate keep strategy

Invalid values are rejected immediately:

```
$ twinhunter find /tmp -k invalid
Error: invalid keep strategy: "invalid" (valid: oldest, newest, shortest)
```

### Dry run before deleting

```sh
twinhunter find /backup -r -d -n
```

### Exclude build artifacts and node_modules

```sh
twinhunter find /project -r -x "*.log,*.tmp" --exclude-dir "dist,build"
```

### Export and clean later

```sh
# Step 1: Find and export
twinhunter find /data -r -o dupes.json

# Step 2: Review the report
cat dupes.json | jq '.duplicate_groups[] | {hash: .Hash, files: [.Files[].Path]}'

# Step 3: Clean with confirmation
twinhunter clean dupes.json -d
```

### Replace with hard links (saves disk space)

```sh
twinhunter find /archive -r -l hard -k newest
```

### Interactive: scan, then choose action

```sh
twinhunter find /photos -r
# After scan, a numbered menu lets you pick delete/link/skip
```

## Building

```sh
# Quick build
go build

# Production build with version info
go build -ldflags="-X 'github.com/mrinjamul/twinhunter/cmd.Version=$(git describe --tags $(git rev-list --tags --max-count=1) || echo "dev")' -X 'github.com/mrinjamul/twinhunter/cmd.GitCommit=$(git rev-parse HEAD)'"

# Run locally
go run . find /path/to/dir -r
```

## Testing

```sh
# All tests
go test ./...

# Verbose
go test ./... -v

# Single package
go test ./core -run TestScanDuplicates -v
```

Tests use the `test_data/` directory for fixtures.

## Project Structure

```
main.go              → entry point
cmd/
  root.go            → root command, dispatches to find
  find.go            → find command with all flags
  clean.go           → clean duplicates from saved report
  completion.go      → shell completion generator
  version.go         → version info
core/
  scanner.go         → file discovery with WalkDir
  hasher.go          → Blake3 + SHA256 hashing pipeline
  dedup.go           → duplicate detection and grouping
  actions.go         → delete, link, backup operations
  report.go          → JSON/CSV/HTML export
filters/
  filters.go         → exclude patterns, size filters
models/
  models.go          → FileInfo, DuplicateGroup, Report structs
test_data/           → test fixtures
```

## License

MIT License — see [LICENSE](LICENSE) for details.

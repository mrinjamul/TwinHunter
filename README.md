# *TwinHunter* (formerly known as `go-dupfinder`)

[![build status](https://github.com/mrinjamul/twinhunter/workflows/test/badge.svg)]()
[![build status](https://github.com/mrinjamul/twinhunter/workflows/release/badge.svg)]()
[![go version](https://img.shields.io/github/go-mod/go-version/mrinjamul/twinhunter.svg)](https://github.com/mrinjamul/twinhunter)
[![GoReportCard](https://goreportcard.com/badge/github.com/mrinjamul/twinhunter)](https://goreportcard.com/report/github.com/mrinjamul/twinhunter)
[![Code style: standard](https://img.shields.io/badge/code%20style-standard-blue.svg)]()
[![License: Apache 2](https://img.shields.io/badge/License-MIT%20-blue.svg)](https://github.com/mrinjamul/twinhunter/blob/master/LICENSE)
[![Github all releases](https://img.shields.io/github/downloads/mrinjamul/twinhunter/total.svg)](https://GitHub.com/mrinjamul/twinhunter/releases/)

Find and clean up duplicate files right from your terminal.

## Features

- **Fast scanning.** Parallel hashing workers powered by WalkDir.
- **Two-stage hashing.** Blake3 for speed, SHA256 to be sure.
- **Flexible filtering.** Min/max size, glob and regex patterns, directory excludes.
- **Smart cleanup.** Delete, hard link, soft link, or move to a backup folder.
- **Multiple outputs.** Pretty terminal, JSON, CSV, or HTML reports.
- **Sortable results.** By size, number of copies, or file path.
- **Keep strategies.** Automatically decide which copy to keep (oldest, newest, shortest path).
- **Dry run.** See what would happen before making any changes.
- **Progress indicator.** Live file count while scanning.
- **Hard link detection.** Know which files already share disk space.
- **Shell completion.** Bash, Zsh, Fish, and PowerShell.

## Quick Start

```sh
# Scan wherever you are right now
twinhunter

# Point at a specific folder
twinhunter /path/to/dir

# Dive into subfolders and see details
twinhunter find /path/to/dir -r -v

# Save results to a JSON file
twinhunter find /path/to/dir -r -o report.json

# Find duplicates and delete them (it will ask first)
twinhunter find /path/to/dir -r -d

# Replace duplicates with hard links to save space
twinhunter find /path/to/dir -r -l hard

# Preview what would happen without touching anything
twinhunter find /path/to/dir -r -d -n

# Skip the "are you sure?" prompts
twinhunter find /path/to/dir -r -d -y

# Skip certain file types like logs and temp files
twinhunter find /path/to/dir -r --exclude-ext "log,tmp"

# Move duplicates to a backup folder instead of deleting
twinhunter find /path/to/dir -r --backup-dir ./dupe_backup
```

## Commands

### `twinhunter [path]`

A quick way to scan the current folder without subdirectories. Just give it an optional path. For all the flags and options, see the `find` command below.

### `twinhunter find [path] [flags]`

All the power, all the flags. Scan any directory with fine-grained control.

| Flag | Short | Description |
|---|---|---|
| `--recursive` | `-r` | Look inside subfolders too |
| `--verbose` | `-v` | Show hashes, sizes, and timestamps for every file in a group |
| `--min-size` | `-m` | Ignore files smaller than this (e.g. `1M`, `100K`, `1G`) |
| `--max-size` | `-M` | Ignore files larger than this (e.g. `10M`) |
| `--exclude` | `-x` | Skip files matching these glob patterns (separate with commas) |
| `--exclude-regex` | | Skip files matching these regex patterns (separate with commas) |
| `--exclude-dir` | | Directory names to skip entirely (defaults: `.git,node_modules,.svn,__pycache__`; overrides defaults if set) |
| `--exclude-ext` | | File extensions to skip (e.g. `log,tmp`) |
| `--workers` | `-w` | How many parallel hashing workers to use (0 = use all CPU cores) |
| `--output` | `-o` | Save a report to file (format is detected from the filename) |
| `--format` | `-f` | How to display results: `pretty`, `json`, `csv`, or `silent` (default: `pretty`) |
| `--sort` | | Order groups by: `size`, `count`, or `path` (default: `size`) |
| `--keep` | `-k` | Which copy to keep: `oldest` (default), `newest`, `shortest` |
| `--link` | `-l` | Replace duplicates with links: `hard` or `soft` |
| `--delete` | `-d` | Delete the duplicate files |
| `--backup-dir` | | Move duplicates to a backup folder instead |
| `--yes` | `-y` | Skip all confirmation prompts |
| `--dry-run` | `-n` | Preview what will happen without changing anything |

### `twinhunter clean report.json`

Run cleanup on a report you saved earlier. Works with JSON, CSV, or HTML exports.

One action flag is **required** (`--delete`, `--link`, or `--backup-dir`).

| Flag | Short | Description |
|---|---|---|
| `--delete` | `-d` | Delete the duplicate files |
| `--link` | `-l` | Replace with links: `hard` or `soft` |
| `--backup-dir` | | Move duplicates to a backup directory |
| `--keep` | `-k` | Which copy to keep: `oldest` (default), `newest`, `shortest` |
| `--dry-run` | `-n` | Preview what will happen without changing anything |

### `twinhunter completion [bash|zsh|fish|powershell]`

Get tab-completion for your favorite shell.

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

TwinHunter can show results in several ways. Pick the one that fits your workflow.

### Pretty (terminal)

```
Duplicate Groups: 1
Duplicate Files:  2
Wasted Space:     190 B
Recoverable:      42.2%

Group 1 - 3 copies, 95 B each
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

Each row is one file with its group, hash, size, path, duplicate status, and modification time.

### Export to File

```sh
# Auto-detects format from extension
twinhunter find /path -r -o report.json
twinhunter find /path -r -o report.csv
twinhunter find /path -r -o report.html
```

The `-o` flag exports to file independently of `-f`. Use both to display CSV/JSON in terminal while saving a different format to file.

## Keep Strategies

Not sure which copy to keep? TwinHunter can decide for you using one of these rules:

| Strategy | Keeps |
|---|---|
| `oldest` | File with earliest modification time (default) |
| `newest` | File with latest modification time |
| `shortest` | File with the shortest file path |

## Interactive Action Prompt

Once the scan finishes, you will see a menu like this:

```
Select action:
  1) Delete duplicates
  2) Replace with hard links
  3) Replace with soft links
  4) Backup duplicates
  5) Skip

Choice [1-5, Enter=5]:
```

Pressing Enter without typing anything skips the action and exits cleanly.
After selecting an action, choose which copy to keep:

```
Keep strategy:
  1) Oldest
  2) Newest
  3) Shortest path

Choice [1-3, Enter=1]:
```

Pressing Enter here picks **Oldest** (the default).

Finally, confirm:

```
Proceed? [y/N]:
```

Pressing Enter says no and exits cleanly. Type `y` or `yes` to go ahead.

Use `-y` to skip all prompts when using `--delete`, `--link`, or `--backup-dir` flags directly.

Interactive menus only show up in `pretty` terminal output. Formats like `json`, `csv`, and `silent` skip all prompts and exit right after reporting results.

## Examples

### Find large duplicates

Only care about big files wasting space?

```sh
twinhunter find /photos -r -m 1M -v
```

### Find and delete, keeping the oldest copy

Clean up old duplicates and keep the earliest version.

```sh
twinhunter find /documents -r -d -k oldest
```

### Find and keep newest, using hard links

Replace copies with hard links and keep the most recent file.

```sh
twinhunter find /downloads -r -l hard -k newest
```

### Sort by number of copies or path

```sh
twinhunter find /media -r --sort count
twinhunter find /media -r --sort path
```

### Silent scan, export to JSON

Great for scripts and automation.

```sh
twinhunter find /data -r -f silent -o dupes.json
```

### Validate keep strategy

Typing an invalid strategy will tell you what is allowed:

```
$ twinhunter find /tmp -k invalid
Error: invalid keep strategy: "invalid" (valid: oldest, newest, shortest)
```

### Dry run before deleting

Preview everything before committing.

```sh
twinhunter find /backup -r -d -n
```

### Exclude build artifacts and node_modules

Skip build artifacts and temp files during the scan.

```sh
twinhunter find /project -r -x "*.log,*.tmp" --exclude-dir "dist,build"
```

### Export and clean later

Save a report, review it, then clean up when you are ready.

```sh
# Step 1: Find and export
twinhunter find /data -r -o dupes.json

# Step 2: Review the report
cat dupes.json | jq '.duplicate_groups[] | {hash: .Hash, files: [.Files[].Path]}'

# Step 3: Clean with confirmation
twinhunter clean dupes.json -d
```

### Replace with hard links (saves disk space)

No more wasted space from identical files sitting in different folders.

```sh
twinhunter find /archive -r -l hard -k newest
```

### Interactive: scan, then choose action

Scan first, decide later. The menu lets you pick delete, link, backup, or skip.

```sh
twinhunter find /photos -r
```

### Backup duplicates to a folder

Move duplicates somewhere safe instead of deleting them outright.

```sh
twinhunter find /docs -r --backup-dir ./dupe_backup -k oldest
```

## Building

```sh
# Compile from source
go build

# Run directly without building first
go run . find /path/to/dir -r
```

## Testing

TwinHunter comes with a full test suite. Run it to make sure everything works:

```sh
# All tests
go test ./...

# See what each test is doing
go test ./... -v

# Run tests for just one package
go test ./core -run TestScanDuplicates -v
```

## Project Structure

```
main.go              → app starts here
cmd/
  root.go            → the main `twinhunter` command, sends you to `find`
  find.go            → the `find` command with all its flags
  clean.go           → the `clean` command for acting on saved reports
  completion.go      → generates shell completion scripts
  version.go         → prints version info
core/
  scanner.go         → walks directories and discovers files
  hasher.go          → hashing pipeline (Blake3 + SHA256)
  dedup.go           → finds and groups duplicates
  actions.go         → delete, link, and backup operations
  report.go          → exports reports as JSON, CSV, or HTML
filters/
  filters.go         → file filtering (exclude patterns, size ranges)
models/
  models.go          → data types (FileInfo, DuplicateGroup, Report)
test_data/           → files used by automated tests
```

## License

MIT License. See [LICENSE](LICENSE) for details.

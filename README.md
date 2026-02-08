# var

A terminal UI for git archeology. Browse commits, trace file history, search for changes, and read blame — without leaving the terminal.

## Install

```bash
go build -o var .
```

## Usage

```bash
./var          # open in current repo
```

`var` opens in **commit list mode**, showing files changed in each commit. Press `Space` to drill into a file's full history in **single-file mode**.

## Features

- **Four display modes** — diff, context (+10 lines), full file, and blame. Cycle with `c`.
- **Pickaxe search** — press `s` to find commits that added or removed a specific string.
- **Reflog traversal** — press `r` to navigate reflog entries instead of commit history.
- **Word-level highlighting** — inline diffs show exactly what changed within each line.
- **Hunk jumping** — `n`/`N` to jump between diff hunks.
- **File filtering** — `/` to fuzzy-filter the file list.

Display modes and commit sources are orthogonal — any display works with any source.

## Keys

### Commit List Mode

| Key | Action |
|-----|--------|
| `j/k` | Navigate files |
| `[/]` | Older/newer commit |
| `Space` | Enter single-file mode |
| `/` | Filter files |
| `n/N` | Next/previous hunk |
| `Tab` | Switch focus |
| `z` | Toggle commit description |
| `q` | Quit |

### Single-File Mode

| Key | Action |
|-----|--------|
| `c` | Cycle display: diff / ctx / full / blame |
| `r` | Toggle reflog source |
| `s` | Pickaxe search |
| `[/]` | Older/newer in current source |
| `d/u` | Half page down/up |
| `n/N` | Next/previous hunk |
| `z` | Toggle commit description |
| `Esc` | Deactivate source / exit mode |
| `1` | Back to commit list |

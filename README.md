# var

A TUI for browsing git commit history and file changes.

## Build

```bash
go build -o var .
```

## Run

```bash
./var
```

## Keyboard Shortcuts

### Mode Switching
- `1` - Switch to commit list mode
- `2` - Switch to single-file mode

### Commit List Mode (`1`)
- `j/k` - Navigate files
- `Space` or `2` - Enter single-file mode
- `[/]` - Navigate commits (older/newer)
- `/` - Filter files
- `n/N` - Jump to next/previous hunk
- `Tab` - Switch focus between sidebar and diff view
- `q` - Quit

### Single-File Mode (`2`)
- `c` - Cycle display: diff → context (+10) → full file → blame
- `r` - Toggle reflog source (navigate reflog entries instead of file commits)
- `s` - Toggle pickaxe search (find commits that added/removed a string)
- `f` - Toggle function log (navigate history of a specific function)
- `[/]` - Navigate history (older/newer) — works with all sources
- `d/u` - Scroll half page down/up
- `n/N` - Jump to next/previous hunk
- `Esc` - Deactivate current source, or exit single-file mode
- `1` or `q` - Back to commit list mode

Display modes and commit sources are orthogonal — any display (diff/ctx/full/blame) works with any source (commits/reflog/pickaxe/function log).

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
- `c` - Cycle view: diff → context (+10) → full file
- `[/]` - Navigate file history (older/newer)
- `d/u` - Scroll half page down/up
- `n/N` - Jump to next/previous hunk
- `1` or `q` - Back to commit list mode

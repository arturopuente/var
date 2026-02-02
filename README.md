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

### Commit Browsing Mode
- `j/k` - Navigate files
- `Space` - Enter single-file mode
- `[/]` - Navigate commits (older/newer)
- `/` - Filter files
- `Tab` - Switch focus between sidebar and diff view
- `q` - Quit

### Single-File Mode
- `1` - Diff view (3 lines context)
- `2` - Context view (10 lines context)
- `3` - Full file view
- `[/]` - Navigate file history (older/newer)
- `d/u` - Scroll half page down/up
- `q` - Back to commit browsing mode

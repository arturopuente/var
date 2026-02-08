Output demo.gif
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set FontSize 14
Set Padding 10

# Launch var
Type "./var"
Enter
Sleep 2s

# Browse commits — back 3
Type "["
Sleep 800ms
Type "["
Sleep 800ms
Type "["
Sleep 800ms

# Forward 3
Type "]"
Sleep 800ms
Type "]"
Sleep 800ms
Type "]"
Sleep 800ms

# Enter single-file mode
Type " "
Sleep 1.5s

# Navigate file history — back 3
Type "["
Sleep 800ms
Type "["
Sleep 800ms
Type "["
Sleep 800ms

# Forward 3
Type "]"
Sleep 800ms
Type "]"
Sleep 800ms
Type "]"
Sleep 800ms

# Scroll down to see diff
Type "d"
Sleep 1s

# Cycle to context view
Type "c"
Sleep 1.5s

# Cycle to full file
Type "c"
Sleep 1.5s

# Cycle to blame
Type "c"
Sleep 1.5s

# Back to diff
Type "c"
Sleep 1s

# Pickaxe search
Type "s"
Sleep 500ms
Type "GetDiff"
Sleep 500ms
Enter
Sleep 2s

# Navigate pickaxe results
Type "["
Sleep 1s
Type "["
Sleep 1s

# Deactivate search
Escape
Sleep 1s

# Exit
Type "q"
Sleep 500ms
Type "q"
Sleep 500ms

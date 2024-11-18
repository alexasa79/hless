# Bugs
1. [x] 'a' adds the selected word to the next search pattern after currently selected and not to the next available search pattern.
2. [x] search colors 4 and 5 are almost identical.
3. [ ] Crash in jump parser, when entering something that starts with a number but is not a proper number.
4. [ ] Fix working with stdin.
5. [x] Except for the cursor, do not print spaces to fill the rest of the line.
6. [ ] Fix selective view for large files.
7. [ ] Crash:
```
Caught panic:

runtime error: slice bounds out of range [:134] with length 129

Restoring terminal...

goroutine 1 [running]:
runtime/debug.Stack()
        /opt/homebrew/Cellar/go/1.23.1/libexec/src/runtime/debug/stack.go:26 +0x5e
runtime/debug.PrintStack()
        /opt/homebrew/Cellar/go/1.23.1/libexec/src/runtime/debug/stack.go:18 +0x13
github.com/charmbracelet/bubbletea.(*Program).Run.func1()
        /Users/sasandl/go/pkg/mod/github.com/charmbracelet/bubbletea@v0.25.0/tea.go:478 +0x8b
panic({0x5d5100?, 0xc000418030?})
        /opt/homebrew/Cellar/go/1.23.1/libexec/src/runtime/panic.go:785 +0x132
github.com/alexasa79/hless/internal/model.(*Model).cegmentize(0xc0003be008, 0xc000b72960, 0xc0003be060, 0xffffffffffffffff, 0xffffffffffffffff, 0xc0003bed40, 0x3?)
        /Users/sasandl/hobbies/hless/internal/model/model.go:1225 +0xb3c
github.com/alexasa79/hless/internal/model.(*Model).View(0xc0003be008)
        /Users/sasandl/hobbies/hless/internal/model/render.go:320 +0x3a5
github.com/charmbracelet/bubbletea.(*Program).eventLoop(0xc0003bb860, {0x63b428?, 0xc0003be008?}, 0xc000020380)
        /Users/sasandl/go/pkg/mod/github.com/charmbracelet/bubbletea@v0.25.0/tea.go:413 +0x61f
github.com/charmbracelet/bubbletea.(*Program).Run(0xc0003bb860)
        /Users/sasandl/go/pkg/mod/github.com/charmbracelet/bubbletea@v0.25.0/tea.go:543 +0x84b
main.main()
        /Users/sasandl/hobbies/hless/hless.go:141 +0x810
```

# Features
1. [x] Support more marker colors.
2. [ ] Select a line and show timestamp relative to that line.
3. [x] Allow adding more characters or words to the selected word with shift+arrows.
4. [x] Show search matches for the current search in bold.
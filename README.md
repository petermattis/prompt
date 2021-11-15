# Prompt [![Go Reference](https://pkg.go.dev/badge/github.com/petermattis/prompt.svg)](https://pkg.go.dev/github.com/petermattis/prompt)

Prompt is a command line prompt editor with history, kill-ring, and tab
completion. It was inspired by linenoise and derivatives which eschew
usage of terminfo/termcap in favor of treating everything like a VT100
terminal. This is taken a bit further with support for additional input
escape sequences that cover ~75% of the terminals in the terminfo database.
A minimal set of output escape sequences is used for rendering the prompt.

// Package cmdclass provides shared command classification maps used by
// hook and daemon packages to decide which commands to skip.
package cmdclass

// TrivialCommands are commands that produce no meaningful output to filter,
// or that would break if wrapped in a pipe.
var TrivialCommands = map[string]bool{
	"cd":     true,
	"pwd":    true,
	"echo":   true,
	"export": true,
	"source": true,
	"which":  true,
	"type":   true,
	"true":   true,
	"false":  true,
	"set":    true,
	"unset":  true,
	"alias":  true,
	"hash":   true,
	"read":   true,
	"return": true,
	"exit":   true,
	"logout": true,
	"exec":   true,
}

// InteractiveCommands are programs that need direct terminal access
// and should not have their output piped through gotk.
var InteractiveCommands = map[string]bool{
	"vim":        true,
	"vi":         true,
	"nvim":       true,
	"nano":       true,
	"emacs":      true,
	"less":       true,
	"more":       true,
	"man":        true,
	"top":        true,
	"htop":       true,
	"btop":       true,
	"watch":      true,
	"ssh":        true,
	"mosh":       true,
	"tmux":       true,
	"screen":     true,
	"fzf":        true,
	"nnn":        true,
	"ranger":     true,
	"mc":         true,
	"lazygit":    true,
	"lazydocker": true,
	"k9s":        true,
}

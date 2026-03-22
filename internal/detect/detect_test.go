package detect

import (
	"testing"

	"github.com/antikkorps/GoTK/internal/filter"
)

func TestIdentify(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    CmdType
	}{
		// Grep family
		{"grep", "grep", CmdGrep},
		{"rg", "rg", CmdGrep},
		{"ag", "ag", CmdGrep},
		{"ack", "ack", CmdGrep},

		// Find family
		{"find", "find", CmdFind},
		{"fd", "fd", CmdFind},

		// Git family
		{"git", "git", CmdGit},
		{"gh", "gh", CmdGit},

		// Go tool
		{"go", "go", CmdGoTool},

		// Ls family
		{"ls", "ls", CmdLs},
		{"exa", "exa", CmdLs},
		{"eza", "eza", CmdLs},
		{"lsd", "lsd", CmdLs},

		// Docker family
		{"docker", "docker", CmdDocker},
		{"docker-compose", "docker-compose", CmdDocker},
		{"podman", "podman", CmdDocker},

		// Npm family
		{"npm", "npm", CmdNpm},
		{"yarn", "yarn", CmdNpm},
		{"pnpm", "pnpm", CmdNpm},
		{"npx", "npx", CmdNpm},
		{"bun", "bun", CmdNpm},

		// Cargo family
		{"cargo", "cargo", CmdCargo},
		{"rustc", "rustc", CmdCargo},

		// Make family
		{"make", "make", CmdMake},
		{"cmake", "cmake", CmdMake},
		{"ninja", "ninja", CmdMake},

		// Curl family
		{"curl", "curl", CmdCurl},
		{"wget", "wget", CmdCurl},

		// Python family
		{"python", "python", CmdPython},
		{"python3", "python3", CmdPython},
		{"pip", "pip", CmdPython},

		// Tree
		{"tree", "tree", CmdTree},

		// Terraform family
		{"terraform", "terraform", CmdTerraform},
		{"tofu", "tofu", CmdTerraform},

		// Kubectl family
		{"kubectl", "kubectl", CmdKubectl},
		{"helm", "helm", CmdKubectl},

		// Jq family
		{"jq", "jq", CmdJq},
		{"yq", "yq", CmdJq},

		// Tar family
		{"tar", "tar", CmdTar},
		{"zip", "zip", CmdTar},
		{"unzip", "unzip", CmdTar},

		// SSH family
		{"ssh", "ssh", CmdSSH},
		{"scp", "scp", CmdSSH},
		{"rsync", "rsync", CmdSSH},

		// Unknown
		{"unknown command", "htop", CmdGeneric},
		{"empty string", "", CmdGeneric},

		// Path-qualified commands
		{"absolute path grep", "/usr/bin/grep", CmdGrep},
		{"absolute path find", "/usr/bin/find", CmdFind},
		{"absolute path git", "/usr/local/bin/git", CmdGit},
		{"absolute path go", "/usr/local/go/bin/go", CmdGoTool},
		{"absolute path ls", "/bin/ls", CmdLs},
		{"absolute path curl", "/usr/bin/curl", CmdCurl},
		{"absolute path unknown", "/usr/bin/htop", CmdGeneric},

		// Windows-style .exe suffix
		{"grep.exe", "grep.exe", CmdGrep},
		{"git.exe", "git.exe", CmdGit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Identify(tt.command)
			if got != tt.want {
				t.Errorf("Identify(%q) = %d, want %d", tt.command, got, tt.want)
			}
		})
	}
}

func TestFiltersFor(t *testing.T) {
	tests := []struct {
		name    string
		cmdType CmdType
		minLen  int // minimum number of filters expected
	}{
		{"grep filters", CmdGrep, 2},
		{"find filters", CmdFind, 2},
		{"git filters", CmdGit, 1},
		{"go filters", CmdGoTool, 2},
		{"ls filters", CmdLs, 1},
		{"docker filters", CmdDocker, 1},
		{"npm filters", CmdNpm, 1},
		{"cargo filters", CmdCargo, 1},
		{"make filters", CmdMake, 1},
		{"curl filters", CmdCurl, 1},
		{"python filters", CmdPython, 1},
		{"tree filters", CmdTree, 1},
		{"terraform filters", CmdTerraform, 1},
		{"kubectl filters", CmdKubectl, 1},
		{"jq filters", CmdJq, 1},
		{"tar filters", CmdTar, 1},
		{"ssh filters", CmdSSH, 1},
		{"generic filters", CmdGeneric, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := FiltersFor(tt.cmdType)
			if len(filters) < tt.minLen {
				t.Errorf("FiltersFor(%d) returned %d filters, want at least %d", tt.cmdType, len(filters), tt.minLen)
			}

			// Verify filters are callable (don't panic)
			for i, f := range filters {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("filter %d panicked: %v", i, r)
						}
					}()
					_ = f("test input")
				}()
			}
		})
	}
}

func TestFiltersForGrepContainsCompressPaths(t *testing.T) {
	filters := FiltersFor(CmdGrep)
	// The first filter for grep should be CompressPaths
	// Test by checking it produces the same result
	input := "test"
	got := filters[0](input)
	want := filter.CompressPaths(input)
	if got != want {
		t.Errorf("first grep filter should be CompressPaths")
	}
}

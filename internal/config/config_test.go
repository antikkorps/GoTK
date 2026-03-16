package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_ReturnsExpectedValues(t *testing.T) {
	cfg := Default()

	// General defaults
	if cfg.General.MaxLines != 50 {
		t.Errorf("MaxLines = %d, want 50", cfg.General.MaxLines)
	}
	if cfg.General.Stats != false {
		t.Error("Stats should default to false")
	}
	if cfg.General.ShellMode != false {
		t.Error("ShellMode should default to false")
	}

	// Filter defaults (all true)
	if !cfg.Filters.StripANSI {
		t.Error("StripANSI should default to true")
	}
	if !cfg.Filters.NormalizeWhitespace {
		t.Error("NormalizeWhitespace should default to true")
	}
	if !cfg.Filters.Dedup {
		t.Error("Dedup should default to true")
	}
	if !cfg.Filters.CompressPaths {
		t.Error("CompressPaths should default to true")
	}
	if !cfg.Filters.TrimDecorative {
		t.Error("TrimDecorative should default to true")
	}
	if !cfg.Filters.Truncate {
		t.Error("Truncate should default to true")
	}

	// Security defaults
	if cfg.Security.CommandTimeout != 30 {
		t.Errorf("CommandTimeout = %d, want 30", cfg.Security.CommandTimeout)
	}
	if cfg.Security.MaxOutputBytes != 10*1024*1024 {
		t.Errorf("MaxOutputBytes = %d, want %d", cfg.Security.MaxOutputBytes, 10*1024*1024)
	}
	if !cfg.Security.RedactSecrets {
		t.Error("RedactSecrets should default to true")
	}

	// Commands should be an empty map, not nil
	if cfg.Commands == nil {
		t.Error("Commands map should not be nil")
	}
	if len(cfg.Commands) != 0 {
		t.Errorf("Commands should be empty, got %d entries", len(cfg.Commands))
	}
}

func TestApplyTOML_GeneralSection(t *testing.T) {
	toml := `[general]
max_lines = 100
stats = true
shell_mode = true
`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.General.MaxLines != 100 {
		t.Errorf("MaxLines = %d, want 100", cfg.General.MaxLines)
	}
	if !cfg.General.Stats {
		t.Error("Stats should be true")
	}
	if !cfg.General.ShellMode {
		t.Error("ShellMode should be true")
	}
}

func TestApplyTOML_FiltersSection(t *testing.T) {
	toml := `[filters]
strip_ansi = false
normalize_whitespace = false
dedup = false
compress_paths = false
trim_decorative = false
truncate = false
`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.Filters.StripANSI {
		t.Error("StripANSI should be false")
	}
	if cfg.Filters.NormalizeWhitespace {
		t.Error("NormalizeWhitespace should be false")
	}
	if cfg.Filters.Dedup {
		t.Error("Dedup should be false")
	}
	if cfg.Filters.CompressPaths {
		t.Error("CompressPaths should be false")
	}
	if cfg.Filters.TrimDecorative {
		t.Error("TrimDecorative should be false")
	}
	if cfg.Filters.Truncate {
		t.Error("Truncate should be false")
	}
}

func TestApplyTOML_SecuritySection(t *testing.T) {
	toml := `[security]
command_timeout = 60
max_output_bytes = 5242880
redact_secrets = false
`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.Security.CommandTimeout != 60 {
		t.Errorf("CommandTimeout = %d, want 60", cfg.Security.CommandTimeout)
	}
	if cfg.Security.MaxOutputBytes != 5242880 {
		t.Errorf("MaxOutputBytes = %d, want 5242880", cfg.Security.MaxOutputBytes)
	}
	if cfg.Security.RedactSecrets {
		t.Error("RedactSecrets should be false")
	}
}

func TestApplyTOML_CommandsSection(t *testing.T) {
	toml := `[commands]
mygrep = "grep"
mybuild = "go"
`
	cfg := Default()
	applyTOML(cfg, toml)

	if v, ok := cfg.Commands["mygrep"]; !ok || v != "grep" {
		t.Errorf("Commands[mygrep] = %q, want %q", v, "grep")
	}
	if v, ok := cfg.Commands["mybuild"]; !ok || v != "go" {
		t.Errorf("Commands[mybuild] = %q, want %q", v, "go")
	}
}

func TestApplyTOML_BooleanValues(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"1", "1", true},
		{"0", "0", false},
		{"yes", "yes", true},
		{"no", "no", false},
		{"random", "random", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBool(tt.val)
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestApplyTOML_IntegerValues(t *testing.T) {
	tests := []struct {
		name     string
		toml     string
		wantMax  int
	}{
		{"positive", "[general]\nmax_lines = 200", 200},
		{"zero", "[general]\nmax_lines = 0", 0},
		{"invalid keeps default", "[general]\nmax_lines = abc", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			applyTOML(cfg, tt.toml)
			if cfg.General.MaxLines != tt.wantMax {
				t.Errorf("MaxLines = %d, want %d", cfg.General.MaxLines, tt.wantMax)
			}
		})
	}
}

func TestApplyTOML_CommentHandling(t *testing.T) {
	tests := []struct {
		name    string
		toml    string
		wantMax int
	}{
		{
			"full line comment",
			"# this is a comment\n[general]\nmax_lines = 75",
			75,
		},
		{
			"inline comment",
			"[general]\nmax_lines = 75 # this is inline",
			75,
		},
		{
			"comment-only lines ignored",
			"[general]\n# max_lines = 999\nmax_lines = 42",
			42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			applyTOML(cfg, tt.toml)
			if cfg.General.MaxLines != tt.wantMax {
				t.Errorf("MaxLines = %d, want %d", cfg.General.MaxLines, tt.wantMax)
			}
		})
	}
}

func TestApplyTOML_EmptyAndBlankLines(t *testing.T) {
	toml := `

[general]

max_lines = 30

[filters]

strip_ansi = false

`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.General.MaxLines != 30 {
		t.Errorf("MaxLines = %d, want 30", cfg.General.MaxLines)
	}
	if cfg.Filters.StripANSI {
		t.Error("StripANSI should be false")
	}
}

func TestApplyTOML_QuotedValues(t *testing.T) {
	toml := `[commands]
mycommand = "grep"
`
	cfg := Default()
	applyTOML(cfg, toml)

	if v := cfg.Commands["mycommand"]; v != "grep" {
		t.Errorf("Commands[mycommand] = %q, want %q", v, "grep")
	}
}

func TestApplyTOML_MalformedTOML(t *testing.T) {
	tests := []struct {
		name string
		toml string
	}{
		{"no equals sign", "[general]\nmax_lines 50"},
		{"unclosed section", "[general\nmax_lines = 50"},
		{"garbage", "!@#$%^&*()"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			applyTOML(cfg, tt.toml) // should not panic
			// After malformed input, values remain at defaults
			if cfg.General.MaxLines != 50 {
				t.Errorf("MaxLines = %d, want 50 (default)", cfg.General.MaxLines)
			}
		})
	}
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	// Change to a temp dir where no gotk.toml exists
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	cfg := Load()

	if cfg.General.MaxLines != 50 {
		t.Errorf("MaxLines = %d, want 50", cfg.General.MaxLines)
	}
	if !cfg.Filters.StripANSI {
		t.Error("StripANSI should default to true")
	}
}

func TestLoad_LocalConfigFile(t *testing.T) {
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	content := `[general]
max_lines = 200
stats = true

[filters]
strip_ansi = false

[security]
command_timeout = 120
redact_secrets = false

[commands]
myls = "ls"
`
	if err := os.WriteFile(filepath.Join(tmp, "gotk.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()

	if cfg.General.MaxLines != 200 {
		t.Errorf("MaxLines = %d, want 200", cfg.General.MaxLines)
	}
	if !cfg.General.Stats {
		t.Error("Stats should be true")
	}
	if cfg.Filters.StripANSI {
		t.Error("StripANSI should be false")
	}
	// Filters not mentioned should keep defaults
	if !cfg.Filters.Dedup {
		t.Error("Dedup should still be true (default)")
	}
	if cfg.Security.CommandTimeout != 120 {
		t.Errorf("CommandTimeout = %d, want 120", cfg.Security.CommandTimeout)
	}
	if cfg.Security.RedactSecrets {
		t.Error("RedactSecrets should be false")
	}
	if v := cfg.Commands["myls"]; v != "ls" {
		t.Errorf("Commands[myls] = %q, want %q", v, "ls")
	}
}

func TestLoad_LocalOverridesGlobal(t *testing.T) {
	// This test verifies the override behavior by creating a local config
	// that overrides defaults. We cannot easily mock the global config path,
	// but we can verify that local config values take effect.
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	localContent := `[general]
max_lines = 999
`
	if err := os.WriteFile(filepath.Join(tmp, "gotk.toml"), []byte(localContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.General.MaxLines != 999 {
		t.Errorf("MaxLines = %d, want 999 (local override)", cfg.General.MaxLines)
	}
}

func TestApplyTOML_MultipleSections(t *testing.T) {
	toml := `[general]
max_lines = 25
stats = true

[filters]
strip_ansi = false
dedup = false

[security]
command_timeout = 10
redact_secrets = false

[commands]
rg = "grep"
`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.General.MaxLines != 25 {
		t.Errorf("MaxLines = %d, want 25", cfg.General.MaxLines)
	}
	if !cfg.General.Stats {
		t.Error("Stats should be true")
	}
	if cfg.Filters.StripANSI {
		t.Error("StripANSI should be false")
	}
	if cfg.Filters.Dedup {
		t.Error("Dedup should be false")
	}
	// Untouched filters keep defaults
	if !cfg.Filters.NormalizeWhitespace {
		t.Error("NormalizeWhitespace should still be true")
	}
	if cfg.Security.CommandTimeout != 10 {
		t.Errorf("CommandTimeout = %d, want 10", cfg.Security.CommandTimeout)
	}
	if cfg.Security.RedactSecrets {
		t.Error("RedactSecrets should be false")
	}
	if v := cfg.Commands["rg"]; v != "grep" {
		t.Errorf("Commands[rg] = %q, want %q", v, "grep")
	}
}

func TestApplyMode_Conservative(t *testing.T) {
	cfg := Default()
	cfg.General.Mode = ModeConservative
	cfg.ApplyMode()

	if cfg.General.MaxLines != 200 {
		t.Errorf("Conservative MaxLines = %d, want 200", cfg.General.MaxLines)
	}
	if cfg.Filters.Truncate {
		t.Error("Conservative should disable Truncate")
	}
	if cfg.Filters.TrimDecorative {
		t.Error("Conservative should disable TrimDecorative")
	}
	// These should remain enabled
	if !cfg.Filters.StripANSI {
		t.Error("Conservative should keep StripANSI enabled")
	}
	if !cfg.Filters.Dedup {
		t.Error("Conservative should keep Dedup enabled")
	}
}

func TestApplyMode_Balanced(t *testing.T) {
	cfg := Default()
	cfg.General.Mode = ModeBalanced
	cfg.ApplyMode()

	// Balanced should not change defaults
	if cfg.General.MaxLines != 50 {
		t.Errorf("Balanced MaxLines = %d, want 50", cfg.General.MaxLines)
	}
	if !cfg.Filters.Truncate {
		t.Error("Balanced should keep Truncate enabled")
	}
}

func TestApplyMode_Aggressive(t *testing.T) {
	cfg := Default()
	cfg.General.Mode = ModeAggressive
	cfg.ApplyMode()

	if cfg.General.MaxLines != 30 {
		t.Errorf("Aggressive MaxLines = %d, want 30", cfg.General.MaxLines)
	}
	if !cfg.Filters.Truncate {
		t.Error("Aggressive should keep Truncate enabled")
	}
	if !cfg.Filters.TrimDecorative {
		t.Error("Aggressive should keep TrimDecorative enabled")
	}
}

func TestApplyMode_EmptyModeIsNoop(t *testing.T) {
	cfg := Default()
	cfg.ApplyMode() // empty mode
	if cfg.General.MaxLines != 50 {
		t.Errorf("Empty mode should not change MaxLines, got %d", cfg.General.MaxLines)
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input string
		want  FilterMode
	}{
		{"conservative", ModeConservative},
		{"CONSERVATIVE", ModeConservative},
		{"balanced", ModeBalanced},
		{"aggressive", ModeAggressive},
		{"Aggressive", ModeAggressive},
		{"unknown", ModeBalanced},
		{"", ModeBalanced},
	}
	for _, tt := range tests {
		got := ParseMode(tt.input)
		if got != tt.want {
			t.Errorf("ParseMode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestApplyTOML_ModeField(t *testing.T) {
	toml := `[general]
mode = "aggressive"
`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.General.Mode != ModeAggressive {
		t.Errorf("Mode = %q, want %q", cfg.General.Mode, ModeAggressive)
	}
}

func TestFindProjectConfig(t *testing.T) {
	// Create a nested dir structure with .gotk.toml at root
	root := t.TempDir()
	// Resolve symlinks (macOS /var → /private/var)
	root, _ = filepath.EvalSymlinks(root)

	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(root, ".gotk.toml")
	if err := os.WriteFile(configPath, []byte("[general]\nmax_lines = 777\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to deeply nested dir
	orig, _ := os.Getwd()
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	found := findProjectConfig()
	if found != configPath {
		t.Errorf("findProjectConfig() = %q, want %q", found, configPath)
	}
}

func TestFindProjectConfig_NotFound(t *testing.T) {
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	found := findProjectConfig()
	if found != "" {
		t.Errorf("findProjectConfig() = %q, want empty string", found)
	}
}

func TestApplyTOML_RulesSection(t *testing.T) {
	toml := `[rules]
always_keep = ["^ERROR:", "^FATAL:"]
always_remove = ["^DEBUG:", "^TRACE:"]
`
	cfg := Default()
	applyTOML(cfg, toml)

	if len(cfg.Rules.AlwaysKeep) != 2 {
		t.Fatalf("AlwaysKeep = %v, want 2 entries", cfg.Rules.AlwaysKeep)
	}
	if cfg.Rules.AlwaysKeep[0] != "^ERROR:" {
		t.Errorf("AlwaysKeep[0] = %q, want ^ERROR:", cfg.Rules.AlwaysKeep[0])
	}
	if len(cfg.Rules.AlwaysRemove) != 2 {
		t.Fatalf("AlwaysRemove = %v, want 2 entries", cfg.Rules.AlwaysRemove)
	}
	if cfg.Rules.AlwaysRemove[0] != "^DEBUG:" {
		t.Errorf("AlwaysRemove[0] = %q, want ^DEBUG:", cfg.Rules.AlwaysRemove[0])
	}
}

func TestApplyTOML_TruncationSection(t *testing.T) {
	toml := `[truncation]
grep = 30
test = 200
git = 100
`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.Truncation["grep"] != 30 {
		t.Errorf("Truncation[grep] = %d, want 30", cfg.Truncation["grep"])
	}
	if cfg.Truncation["test"] != 200 {
		t.Errorf("Truncation[test] = %d, want 200", cfg.Truncation["test"])
	}
	if cfg.Truncation["git"] != 100 {
		t.Errorf("Truncation[git] = %d, want 100", cfg.Truncation["git"])
	}
}

func TestMaxLinesForCommand(t *testing.T) {
	cfg := Default()
	cfg.Truncation["grep"] = 30
	cfg.Truncation["test"] = 200

	if got := cfg.MaxLinesForCommand("grep"); got != 30 {
		t.Errorf("MaxLinesForCommand(grep) = %d, want 30", got)
	}
	if got := cfg.MaxLinesForCommand("test"); got != 200 {
		t.Errorf("MaxLinesForCommand(test) = %d, want 200", got)
	}
	if got := cfg.MaxLinesForCommand("unknown"); got != 50 {
		t.Errorf("MaxLinesForCommand(unknown) = %d, want 50 (default)", got)
	}
}

func TestParseTOMLArray(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"normal", `["a", "b", "c"]`, 3},
		{"single", `["one"]`, 1},
		{"empty array", `[]`, 0},
		{"not array", `hello`, 0},
		{"empty string", ``, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTOMLArray(tt.input)
			if len(got) != tt.want {
				t.Errorf("parseTOMLArray(%q) returned %d items, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestParseProfile(t *testing.T) {
	tests := []struct {
		input string
		want  LLMProfile
	}{
		{"claude", ProfileClaude},
		{"Claude", ProfileClaude},
		{"gpt", ProfileGPT},
		{"openai", ProfileGPT},
		{"chatgpt", ProfileGPT},
		{"gemini", ProfileGemini},
		{"google", ProfileGemini},
		{"unknown", ProfileNone},
		{"", ProfileNone},
	}
	for _, tt := range tests {
		got := ParseProfile(tt.input)
		if got != tt.want {
			t.Errorf("ParseProfile(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestApplyProfile_Claude(t *testing.T) {
	cfg := Default()
	cfg.Profile = ProfileClaude
	cfg.ApplyProfile()

	if cfg.General.MaxLines != 80 {
		t.Errorf("Claude MaxLines = %d, want 80", cfg.General.MaxLines)
	}
	if cfg.General.Mode != ModeBalanced {
		t.Errorf("Claude Mode = %q, want balanced", cfg.General.Mode)
	}
	if cfg.Truncation["grep"] != 120 {
		t.Errorf("Claude grep truncation = %d, want 120", cfg.Truncation["grep"])
	}
}

func TestApplyProfile_GPT(t *testing.T) {
	cfg := Default()
	cfg.Profile = ProfileGPT
	cfg.ApplyProfile()

	if cfg.General.MaxLines != 50 {
		t.Errorf("GPT MaxLines = %d, want 50", cfg.General.MaxLines)
	}
	if cfg.Truncation["grep"] != 80 {
		t.Errorf("GPT grep truncation = %d, want 80", cfg.Truncation["grep"])
	}
}

func TestApplyProfile_Gemini(t *testing.T) {
	cfg := Default()
	cfg.Profile = ProfileGemini
	cfg.ApplyProfile()

	if cfg.General.MaxLines != 150 {
		t.Errorf("Gemini MaxLines = %d, want 150", cfg.General.MaxLines)
	}
	if cfg.General.Mode != ModeConservative {
		t.Errorf("Gemini Mode = %q, want conservative", cfg.General.Mode)
	}
}

func TestApplyProfile_DoesNotOverrideExplicitTruncation(t *testing.T) {
	cfg := Default()
	cfg.Truncation["grep"] = 42 // user set explicitly
	cfg.Profile = ProfileClaude
	cfg.ApplyProfile()

	if cfg.Truncation["grep"] != 42 {
		t.Errorf("explicit grep truncation should be preserved, got %d", cfg.Truncation["grep"])
	}
}

func TestApplyTOML_ProfileSection(t *testing.T) {
	toml := `[profile]
name = "claude"
`
	cfg := Default()
	applyTOML(cfg, toml)

	if cfg.Profile != ProfileClaude {
		t.Errorf("Profile = %q, want claude", cfg.Profile)
	}
}

func TestLoad_ProjectConfigPrecedence(t *testing.T) {
	// Setup: project root has .gotk.toml, subdir has gotk.toml
	root := t.TempDir()
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Project config sets max_lines=150
	projectCfg := "[general]\nmax_lines = 150\n"
	if err := os.WriteFile(filepath.Join(root, ".gotk.toml"), []byte(projectCfg), 0644); err != nil {
		t.Fatal(err)
	}

	// Local config sets max_lines=300 (should win)
	localCfg := "[general]\nmax_lines = 300\n"
	if err := os.WriteFile(filepath.Join(subdir, "gotk.toml"), []byte(localCfg), 0644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	cfg := Load()
	if cfg.General.MaxLines != 300 {
		t.Errorf("MaxLines = %d, want 300 (local should override project)", cfg.General.MaxLines)
	}
}

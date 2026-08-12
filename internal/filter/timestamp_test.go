package filter

import "testing"

func TestStripTimestamps(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "astro build clock prefix",
			input: "12:35:02 [build] output: \"static\"\n" +
				"12:35:02 [build] mode: \"static\"\n" +
				"12:35:03 [vite] ✓ built in 1.08s\n",
			want: "[build] output: \"static\"\n" +
				"[build] mode: \"static\"\n" +
				"[vite] ✓ built in 1.08s\n",
		},
		{
			name: "nested indentation is preserved",
			input: "12:35:03 ✓ Completed in 214ms.\n" +
				"12:35:03   ├─ /fr/index.html (+97ms)\n" +
				"12:35:03   ├─ /index.html (+37ms)\n",
			want: "✓ Completed in 214ms.\n" +
				"  ├─ /fr/index.html (+97ms)\n" +
				"  ├─ /index.html (+37ms)\n",
		},
		{
			name: "bracketed clock",
			input: "[10:10:26] starting\n" +
				"[10:10:27] compiling\n" +
				"[10:10:28] done\n",
			want: "starting\ncompiling\ndone\n",
		},
		{
			name: "iso 8601 with millis and zone",
			input: "2026-08-12T10:10:26.575Z npm error code ERESOLVE\n" +
				"2026-08-12T10:10:26.576Z npm error ERESOLVE could not resolve\n" +
				"2026-08-12T10:10:26.577Z npm error While resolving: @astrojs/tailwind\n",
			want: "npm error code ERESOLVE\n" +
				"npm error ERESOLVE could not resolve\n" +
				"npm error While resolving: @astrojs/tailwind\n",
		},
		{
			name: "iso date with space separator and offset",
			input: "2026-08-12 10:10:26+02:00 worker started\n" +
				"2026-08-12 10:10:27+02:00 worker ready\n" +
				"2026-08-12 10:10:28+02:00 worker done\n",
			want: "worker started\nworker ready\nworker done\n",
		},
		{
			name: "error and warning content survives untouched",
			input: "12:35:02 ERROR: build failed at src/main.ts:42\n" +
				"12:35:02 warning: unused import\n" +
				"12:35:02   at Object.<anonymous> (src/main.ts:42:9)\n",
			want: "ERROR: build failed at src/main.ts:42\n" +
				"warning: unused import\n" +
				"  at Object.<anonymous> (src/main.ts:42:9)\n",
		},
		{
			name:  "below threshold is left alone",
			input: "12:35:02 only one timestamped line\nplain line\nanother plain line\n",
			want:  "12:35:02 only one timestamped line\nplain line\nanother plain line\n",
		},
		{
			name: "mid-line clock is never touched",
			input: "frame= 100 fps=25 time=00:00:04.00 bitrate=1.2\n" +
				"frame= 200 fps=25 time=00:00:08.00 bitrate=1.2\n" +
				"frame= 300 fps=25 time=00:00:12.00 bitrate=1.2\n",
			want: "frame= 100 fps=25 time=00:00:04.00 bitrate=1.2\n" +
				"frame= 200 fps=25 time=00:00:08.00 bitrate=1.2\n" +
				"frame= 300 fps=25 time=00:00:12.00 bitrate=1.2\n",
		},
		{
			// A bare clock never matches (the pattern requires a separator),
			// and a clock followed by only whitespace hits the blank-line guard.
			name: "timestamp-only lines are kept",
			input: "12:35:02 first\n" +
				"12:35:03 second\n" +
				"12:35:04 third\n" +
				"12:35:05\n" +
				"12:35:06   \n",
			want: "first\n" +
				"second\n" +
				"third\n" +
				"12:35:05\n" +
				"12:35:06   \n",
		},
		{
			name: "indentation before the timestamp is preserved",
			input: "  12:35:02 nested one\n" +
				"  12:35:03 nested two\n" +
				"  12:35:04 nested three\n",
			want: "  nested one\n  nested two\n  nested three\n",
		},
		{
			name:  "untimestamped lines pass through",
			input: "12:35:02 first\nno timestamp here\n12:35:03 second\n12:35:04 third\n",
			want:  "first\nno timestamp here\nsecond\nthird\n",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name: "not a clock",
			input: "1234:56:78 nope\n" +
				"1234:56:78 nope\n" +
				"1234:56:78 nope\n",
			want: "1234:56:78 nope\n" +
				"1234:56:78 nope\n" +
				"1234:56:78 nope\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripTimestamps(tt.input)
			if got != tt.want {
				t.Errorf("StripTimestamps(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestStripTimestampsFeedsDedup covers the reason the filter runs before
// Dedup: lines that differ only by their clock cannot collapse until the
// prefix is gone.
func TestStripTimestampsFeedsDedup(t *testing.T) {
	input := "12:35:02 waiting for compiler\n" +
		"12:35:03 waiting for compiler\n" +
		"12:35:04 waiting for compiler\n" +
		"12:35:05 waiting for compiler\n"

	if got := Dedup(input); got != input {
		t.Fatalf("precondition failed: Dedup already collapsed timestamped lines: %q", got)
	}

	got := Dedup(StripTimestamps(input))
	want := "waiting for compiler\n  ... (3 duplicate lines)\n"
	if got != want {
		t.Errorf("Dedup(StripTimestamps(input)) = %q, want %q", got, want)
	}
}

// TestStreamFilterStripsTimestamps mirrors TestStripTimestamps for the
// line-at-a-time path. Streaming cannot look ahead, so the first
// minTimestampLines-1 prefixed lines are emitted with their clock intact and
// stripping starts once the threshold is reached.
func TestStreamFilterStripsTimestamps(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{StripTimestamps: true})

	in := []string{
		"12:35:02 [build] one",
		"12:35:03 [build] two",
		"12:35:04 [build] three",
		"12:35:05   ├─ nested",
		"plain line",
		"12:35:06 ERROR: boom",
	}
	want := []string{
		"12:35:02 [build] one",
		"12:35:03 [build] two",
		"[build] three",
		"  ├─ nested",
		"plain line",
		"ERROR: boom",
	}

	for i, line := range in {
		got, emit := sf.ProcessLine(line)
		if !emit {
			t.Fatalf("line %d (%q) was not emitted", i, line)
		}
		if got != want[i] {
			t.Errorf("ProcessLine(%q) = %q, want %q", line, got, want[i])
		}
	}
}

// TestStreamFilterTimestampsDisabled guards the config toggle.
func TestStreamFilterTimestampsDisabled(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{StripTimestamps: false})

	for range 4 {
		got, _ := sf.ProcessLine("12:35:02 [build] one")
		if got != "12:35:02 [build] one" {
			t.Fatalf("timestamp stripped while disabled: %q", got)
		}
	}
}

package capsule

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple lowercase",
			input: "Hello World",
			want:  "hello world",
		},
		{
			name:  "trim whitespace",
			input: "  hello  ",
			want:  "hello",
		},
		{
			name:  "collapse internal whitespace",
			input: "hello    world",
			want:  "hello world",
		},
		{
			name:  "mixed case with extra spaces",
			input: "  Hello   WORLD  ",
			want:  "hello world",
		},
		{
			name:  "tabs and newlines",
			input: "hello\t\n  world",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \t\n   ",
			want:  "",
		},
		{
			name:  "unicode characters",
			input: "  HÉLLO   WÖRLD  ",
			want:  "héllo wörld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "ascii only",
			input: "hello",
			want:  5,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "unicode - emoji",
			input: "hello 👋",
			want:  7, // 5 letters + 1 space + 1 emoji (emoji is 4 bytes but 1 rune)
		},
		{
			name:  "unicode - chinese characters",
			input: "你好世界",
			want:  4, // 4 characters, each is 3 bytes but 1 rune
		},
		{
			name:  "unicode - mixed",
			input: "hello世界",
			want:  7, // 5 ascii + 2 chinese
		},
		{
			name:  "unicode - accented",
			input: "café",
			want:  4, // 4 characters
		},
		{
			name:  "bytes vs runes verification",
			input: "日本語", // 3 characters, 9 bytes
			want:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountChars(tt.input)
			if got != tt.want {
				t.Errorf("CountChars(%q) = %d, want %d (len=%d bytes)", tt.input, got, tt.want, len(tt.input))
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "single word",
			input: "hello",
			want:  2, // ceil(1 * 1.3) = 2
		},
		{
			name:  "two words",
			input: "hello world",
			want:  3, // ceil(2 * 1.3) = 3
		},
		{
			name:  "ten words",
			input: "one two three four five six seven eight nine ten",
			want:  13, // ceil(10 * 1.3) = 13
		},
		{
			name:  "with extra whitespace",
			input: "  hello   world  ",
			want:  3, // still 2 words, ceil(2 * 1.3) = 3
		},
		{
			name:  "with newlines and tabs",
			input: "hello\nworld\tthere",
			want:  4, // ceil(3 * 1.3) = 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

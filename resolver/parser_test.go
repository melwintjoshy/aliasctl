package resolver

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Command
	}{
		{
			name:  "simple command",
			input: "kubectl",
			expected: Command{
				Name: "kubectl",
				Args: []string{},
			},
		},
		{
			name:  "command with arguments",
			input: "kubectl get pods -n app-prod",
			expected: Command{
				Name: "kubectl",
				Args: []string{
					"get",
					"pods",
					"-n",
					"app-prod",
				},
			},
		},
		{
			name:  "double quoted argument",
			input: `git commit -m "initial commit"`,
			expected: Command{
				Name: "git",
				Args: []string{
					"commit",
					"-m",
					"initial commit",
				},
			},
		},
		{
			name:  "single quoted argument",
			input: `echo 'hello world'`,
			expected: Command{
				Name: "echo",
				Args: []string{
					"hello world",
				},
			},
		},
		{
			name:  "multiple spaces",
			input: "kubectl   get    pods",
			expected: Command{
				Name: "kubectl",
				Args: []string{
					"get",
					"pods",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCommand(tt.input)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf(
					"expected %+v, got %+v",
					tt.expected,
					got,
				)
			}
		})
	}
}

func TestParseCommandErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty command",
			input: "",
		},
		{
			name:  "unclosed single quote",
			input: "echo 'hello",
		},
		{
			name:  "unclosed double quote",
			input: `echo "hello`,
		},
		{
			name:  "unfinished escape",
			input: `echo hello\`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCommand(tt.input)

			if err == nil {
				t.Fatalf(
					"expected error for input %q",
					tt.input,
				)
			}
		})
	}
}

func TestTokenizeBackslashes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "backslash is literal in single quotes",
			input:    `'a\b'`,
			expected: []string{`a\b`},
		},
		{
			name:     "backslash before n is kept in double quotes",
			input:    `"\n"`,
			expected: []string{`\n`},
		},
		{
			name:     "dollar is escapable in double quotes",
			input:    `"\$x"`,
			expected: []string{`$x`},
		},
		{
			name:     "backslash is escapable in double quotes",
			input:    `"\\"`,
			expected: []string{`\`},
		},
		{
			name:     "double quote is escapable in double quotes",
			input:    `"say \"hi\""`,
			expected: []string{`say "hi"`},
		},
		{
			name:     "backtick is escapable in double quotes",
			input:    "\"\\`\"",
			expected: []string{"`"},
		},
		{
			name:     "escaped newline in double quotes is a continuation",
			input:    "\"a\\\nb\"",
			expected: []string{"ab"},
		},
		{
			name:     "escaped space outside quotes joins words",
			input:    `a\ b`,
			expected: []string{"a b"},
		},
		{
			name:     "printf format keeps its escape",
			input:    `printf "[%s]\n" x`,
			expected: []string{"printf", `[%s]\n`, "x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestTokenizeTrailingBackslashInQuotes(t *testing.T) {
	for _, input := range []string{`'a\`, `"a\`} {
		if _, err := tokenize(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestCheckShellSyntax(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "plain command", input: "kubectl get pods -n app-prod"},
		{name: "placeholder", input: "kubectl -n ${NAMESPACE}"},
		{name: "quoted metacharacters", input: `git commit -m "fix: a; b | c"`},
		{name: "single quoted glob", input: `echo '*.go'`},
		{name: "escaped pipe", input: `echo a \| b`},
		{name: "tilde inside a word", input: "git reset HEAD~1"},
		{name: "glob", input: "ls *.go", wantErr: true},
		{name: "pipe", input: "ps aux | grep go", wantErr: true},
		{name: "redirect", input: "echo hi > out", wantErr: true},
		{name: "command chain", input: "make && make test", wantErr: true},
		{name: "bare variable", input: "echo $HOME", wantErr: true},
		{name: "leading tilde", input: "ls ~/src", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkShellSyntax(tt.input)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}

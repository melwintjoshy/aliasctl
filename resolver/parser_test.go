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

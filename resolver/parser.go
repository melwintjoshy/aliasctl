package resolver

import (
	"fmt"
	"strings"
)

func parseCommand(input string) (Command, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return Command{}, err
	}

	if len(tokens) == 0 {
		return Command{}, fmt.Errorf("command cannot be empty")
	}

	return Command{
		Name: tokens[0],
		Args: tokens[1:],
	}, nil
}

// characters bash would treat as syntax; aliases are structured commands, so they'd go through literally
const shellSyntax = "|&;<>()$`*?["

func tokenize(input string) ([]string, error) {
	return scanTokens(input, false)
}

// placeholders are stood in for first since they're substituted, not shell syntax
func checkShellSyntax(input string) error {
	_, err := scanTokens(placeholderPattern.ReplaceAllString(input, "x"), true)
	return err
}

// posix quoting: nothing escapes in single quotes, only $ ` " \ and newline in double quotes
func scanTokens(input string, rejectShellSyntax bool) ([]string, error) {
	var tokens []string
	var current []rune

	var quote rune
	tokenStarted := false

	runes := []rune(input)

	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if quote == '\'' {
			if ch == quote {
				quote = 0
			} else {
				current = append(current, ch)
			}

			continue
		}

		if quote == '"' {
			switch {
			case ch == quote:
				quote = 0

			case ch == '\\' && i+1 < len(runes) && strings.ContainsRune("$`\"\\\n", runes[i+1]):
				i++

				// an escaped newline is a line continuation and disappears
				if runes[i] != '\n' {
					current = append(current, runes[i])
				}

			default:
				current = append(current, ch)
			}

			continue
		}

		switch ch {
		case '\\':
			if i+1 >= len(runes) {
				return nil, fmt.Errorf("unfinished escape sequence")
			}

			i++
			current = append(current, runes[i])
			tokenStarted = true

		case '\'', '"':
			quote = ch
			tokenStarted = true

		case ' ', '\t', '\n':
			if tokenStarted {
				tokens = append(tokens, string(current))
				current = nil
				tokenStarted = false
			}

		default:
			if rejectShellSyntax && (strings.ContainsRune(shellSyntax, ch) || (ch == '~' && !tokenStarted)) {
				return nil, fmt.Errorf(
					"shell syntax %q is not supported in aliases; quote it or use a function",
					string(ch),
				)
			}

			current = append(current, ch)
			tokenStarted = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote")
	}

	if tokenStarted {
		tokens = append(tokens, string(current))
	}

	return tokens, nil
}

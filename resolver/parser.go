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

// posix quoting: nothing escapes in single quotes, only $ ` " \ and newline in double quotes
func tokenize(input string) ([]string, error) {
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

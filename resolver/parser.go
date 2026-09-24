package resolver

import "fmt"

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

func tokenize(input string) ([]string, error) {
	var tokens []string
	var current []rune

	var quote rune
	escaped := false
	tokenStarted := false

	for _, ch := range input {
		if escaped {
			current = append(current, ch)
			escaped = false
			tokenStarted = true
			continue
		}

		if ch == '\\' {
			escaped = true
			tokenStarted = true
			continue
		}

		if quote != 0 {
			if ch == quote {
				quote = 0
			} else {
				current = append(current, ch)
			}

			tokenStarted = true
			continue
		}

		switch ch {
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

	if escaped {
		return nil, fmt.Errorf("unfinished escape sequence")
	}

	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote")
	}

	if tokenStarted {
		tokens = append(tokens, string(current))
	}

	return tokens, nil
}

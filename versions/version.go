package versions

import (
	"regexp"
	"strconv"
	"strings"
)

// Version is a dotted numeric version such as 1.22.4.
type Version []int

var (
	dottedPattern = regexp.MustCompile(`\d+(\.\d+)+`)
	numberPattern = regexp.MustCompile(`\d+`)
)

// Parse finds the first version in tool output like "go1.22.4" or "Client Version: v1.30.2".
func Parse(text string) (Version, bool) {
	// dotted first so "python3 3.12.1" yields 3.12.1, not the 3 from the name
	match := dottedPattern.FindString(text)
	if match == "" {
		match = numberPattern.FindString(text)
	}

	if match == "" {
		return nil, false
	}

	return parseDotted(match)
}

func parseDotted(text string) (Version, bool) {
	parts := strings.Split(text, ".")
	version := make(Version, 0, len(parts))

	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return nil, false
		}

		version = append(version, number)
	}

	return version, true
}

// Compare treats missing segments as zero, so 1.22 equals 1.22.0.
func Compare(a, b Version) int {
	for i := 0; i < max(len(a), len(b)); i++ {
		var left, right int

		if i < len(a) {
			left = a[i]
		}

		if i < len(b) {
			right = b[i]
		}

		if left != right {
			if left < right {
				return -1
			}

			return 1
		}
	}

	return 0
}

func (v Version) String() string {
	parts := make([]string, len(v))

	for i, number := range v {
		parts[i] = strconv.Itoa(number)
	}

	return strings.Join(parts, ".")
}

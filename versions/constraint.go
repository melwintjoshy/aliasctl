package versions

import (
	"fmt"
	"slices"
	"strings"
)

// Constraint is a set of rules that must all hold, e.g. ">=1.29 <1.31".
type Constraint struct {
	rules []rule
}

type rule struct {
	operator string
	version  Version
}

var operators = []string{">=", "<=", ">", "<", "="}

// ParseConstraint accepts "*", a bare version (prefix match) or space-separated comparisons.
// A leading v on any version is ignored.
func ParseConstraint(text string) (Constraint, error) {
	fields := strings.Fields(text)

	if len(fields) == 0 {
		return Constraint{}, fmt.Errorf("version rule is empty")
	}

	if len(fields) == 1 && fields[0] == "*" {
		return Constraint{}, nil
	}

	var constraint Constraint

	for _, field := range fields {
		parsed, err := parseRule(field)
		if err != nil {
			return Constraint{}, err
		}

		constraint.rules = append(constraint.rules, parsed)
	}

	return constraint, nil
}

func parseRule(field string) (rule, error) {
	operator := ""

	for _, candidate := range operators {
		if strings.HasPrefix(field, candidate) {
			operator = candidate
			break
		}
	}

	text := strings.TrimPrefix(field, operator)

	if operator != "" && text == "" {
		return rule{}, fmt.Errorf("operator %q must be followed by a version, e.g. %s1.29", operator, operator)
	}

	// kubectl, helm and node print a leading v, so rules get written that way too
	version, ok := parseDotted(strings.TrimPrefix(text, "v"))
	if !ok {
		return rule{}, fmt.Errorf("invalid version rule %q", field)
	}

	return rule{operator: operator, version: version}, nil
}

func (c Constraint) Match(version Version) bool {
	for _, r := range c.rules {
		if !r.match(version) {
			return false
		}
	}

	return true
}

func (r rule) match(version Version) bool {
	switch r.operator {
	case "":
		// a bare version pins only the segments it names, so 1.22 matches 1.22.x
		return len(version) >= len(r.version) && slices.Equal(version[:len(r.version)], r.version)
	case "=":
		return Compare(version, r.version) == 0
	case ">=":
		return Compare(version, r.version) >= 0
	case ">":
		return Compare(version, r.version) > 0
	case "<=":
		return Compare(version, r.version) <= 0
	case "<":
		return Compare(version, r.version) < 0
	}

	return false
}

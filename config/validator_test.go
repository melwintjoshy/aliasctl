package config

import "testing"

func TestValidateRequiresName(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]string{
			"k": "kubectl",
		},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("expected validation error")
	}

	expected := "name is required"

	if err.Error() != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			err.Error(),
		)
	}
}

func TestValidateRequiresAlias(t *testing.T) {
	cfg := &Config{
		Name: "test",
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("expected validation error")
	}

	expected := "at least one alias or function is required"

	if err.Error() != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			err.Error(),
		)
	}
}

func TestValidateRejectsInvalidAlias(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Aliases: map[string]string{
			"hello world": "echo hello",
		},
	}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsInvalidFunctionName(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Functions: map[string]string{
			"hello;rm": "echo hello",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid function name error")
	}
}

func TestValidateAcceptsValidFunctionName(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Aliases: map[string]string{
			"k": "kubectl",
		},
		Functions: map[string]string{
			"deploy_prod": "echo deploy",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAcceptsValidVariableName(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Variables: map[string]string{
			"NAMESPACE":     "app-prod",
			"ENVIRONMENT_2": "development",
		},
		Aliases: map[string]string{
			"hello": "echo hello",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsInvalidVariableName(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Variables: map[string]string{
			"BAD-NAME": "value",
		},
		Aliases: map[string]string{
			"hello": "echo hello",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid variable name error")
	}
}

func TestValidateAcceptsFunctionsOnly(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Functions: map[string]string{
			"deploy": "echo deploy",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsAliasFunctionCollision(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Aliases: map[string]string{
			"deploy": "echo alias",
		},
		Functions: map[string]string{
			"deploy": "echo function",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected collision error")
	}

	expected := `"deploy" is defined as both an alias and a function`

	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateAcceptsFishFunctionsOnly(t *testing.T) {
	cfg := &Config{
		Name: "test",
		FunctionsFish: map[string]string{
			"deploy": "echo deploy",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsFishFunctionAliasCollision(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Aliases: map[string]string{
			"deploy": "echo alias",
		},
		FunctionsFish: map[string]string{
			"deploy": "echo fish",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected collision error")
	}
}

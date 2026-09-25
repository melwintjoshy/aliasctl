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

func TestValidateTools(t *testing.T) {
	tests := []struct {
		name    string
		tools   map[string]ToolSpec
		wantErr string
	}{
		{
			name: "valid rules",
			tools: map[string]ToolSpec{
				"go":             {Version: "1.22"},
				"kubectl":        {Version: ">=1.29 <1.31"},
				"docker-compose": {Version: "*"},
				"python3":        {Version: "3.12", Check: "python3 --version"},
			},
		},
		{
			name:    "bad rule",
			tools:   map[string]ToolSpec{"go": {Version: "~>1.2"}},
			wantErr: `invalid tool "go": invalid version rule "~>1.2"`,
		},
		{
			name:    "missing rule",
			tools:   map[string]ToolSpec{"go": {Check: "go version"}},
			wantErr: `invalid tool "go": version rule is empty`,
		},
		{
			name:    "bad name",
			tools:   map[string]ToolSpec{"-go": {Version: "1"}},
			wantErr: `invalid tool name: "-go"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Name:    "test",
				Aliases: map[string]string{"k": "kubectl"},
				Tools:   tt.tools,
			}

			err := cfg.Validate()

			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("expected %q, got %v", tt.wantErr, err)
			}
		})
	}
}

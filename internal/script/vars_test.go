package script

import (
	"strings"
	"testing"
)

func TestParseCLIVars(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "simple pair",
			args: []string{"file=report.txt"},
			want: map[string]string{"file": "report.txt"},
		},
		{
			name: "multiple pairs",
			args: []string{"file=a.txt", "style=terse"},
			want: map[string]string{"file": "a.txt", "style": "terse"},
		},
		{
			name: "value with equals",
			args: []string{"url=http://example.com/a=b"},
			want: map[string]string{"url": "http://example.com/a=b"},
		},
		{
			name:    "no equals sign",
			args:    []string{"noequals"},
			wantErr: true,
		},
		{
			name:    "invalid key",
			args:    []string{"123bad=value"},
			wantErr: true,
		},
		{
			name: "underscore key",
			args: []string{"my_file=test.txt"},
			want: map[string]string{"my_file": "test.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCLIVars(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCLIVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for k, v := range tt.want {
					if got[k] != v {
						t.Errorf("got[%q] = %q, want %q", k, got[k], v)
					}
				}
			}
		})
	}
}

func TestBindVars(t *testing.T) {
	tests := []struct {
		name     string
		declared map[string]VarDecl
		supplied map[string]string
		want     map[string]string
		wantErr  bool
	}{
		{
			name: "required provided",
			declared: map[string]VarDecl{
				"file": {Required: true},
			},
			supplied: map[string]string{"file": "report.txt"},
			want:     map[string]string{"file": "report.txt"},
		},
		{
			name: "required missing",
			declared: map[string]VarDecl{
				"file": {Required: true},
			},
			supplied: map[string]string{},
			wantErr:  true,
		},
		{
			name: "default used",
			declared: map[string]VarDecl{
				"style": {Required: false, Default: "verbose"},
			},
			supplied: map[string]string{},
			want:     map[string]string{"style": "verbose"},
		},
		{
			name: "default overridden",
			declared: map[string]VarDecl{
				"style": {Required: false, Default: "verbose"},
			},
			supplied: map[string]string{"style": "terse"},
			want:     map[string]string{"style": "terse"},
		},
		{
			name:     "undeclared var supplied",
			declared: map[string]VarDecl{},
			supplied: map[string]string{"extra": "value"},
			wantErr:  true,
		},
		{
			name:     "no vars no supplied",
			declared: map[string]VarDecl{},
			supplied: map[string]string{},
			want:     map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BindVars(tt.declared, tt.supplied)
			if (err != nil) != tt.wantErr {
				t.Errorf("BindVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for k, v := range tt.want {
					if got[k] != v {
						t.Errorf("got[%q] = %q, want %q", k, got[k], v)
					}
				}
			}
		})
	}
}

func TestBindVars_UndeclaredVarWithNonEmptyDeclared(t *testing.T) {
	// Supplied var not in declared when declared is non-empty → line 49 error
	declared := map[string]VarDecl{
		"style": {Default: "terse"},
	}
	supplied := map[string]string{"extra": "value"}
	_, err := BindVars(declared, supplied)
	if err == nil {
		t.Fatal("expected error for undeclared variable, got nil")
	}
	if !strings.Contains(err.Error(), "undeclared") {
		t.Errorf("expected 'undeclared' in error, got %q", err.Error())
	}
}

func TestBindVars_CaseInsensitiveMatch(t *testing.T) {
	// Declared key is "Name", supplied key is "NAME" — should match case-insensitively
	declared := map[string]VarDecl{
		"Name": {Required: true},
	}
	supplied := map[string]string{"NAME": "Alice"}
	got, err := BindVars(declared, supplied)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["Name"] != "Alice" {
		t.Errorf("expected got['Name']='Alice', got %q", got["Name"])
	}
}

func TestBindVars_UndeclaredVarNilDeclared(t *testing.T) {
	// Supplied var not in declared (nil declared map)
	supplied := map[string]string{"extra": "value"}
	_, err := BindVars(nil, supplied)
	if err == nil {
		t.Fatal("expected error for undeclared variable with nil declared map, got nil")
	}
	if !strings.Contains(err.Error(), "undeclared") {
		t.Errorf("expected 'undeclared' in error, got %q", err.Error())
	}
}

func TestBindVars_HasKeyReturnsFalse(t *testing.T) {
	// hasKey returns false when the key is not present → BindVars uses default
	declared := map[string]VarDecl{
		"style": {Default: "terse"},
	}
	// Supply nothing — hasKey will be called with "style" on empty map and return false
	supplied := map[string]string{}
	got, err := BindVars(declared, supplied)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["style"] != "terse" {
		t.Errorf("expected default 'terse', got %q", got["style"])
	}
}

func TestBindVars_EmptyStringValueSupplied(t *testing.T) {
	// Supplied var exists with empty string value → hasKey returns true, value is ""
	// This covers the hasKey return true branch
	declared := map[string]VarDecl{
		"style": {Default: "terse"},
	}
	// Supply "style" with empty string value
	supplied := map[string]string{"style": ""}
	got, err := BindVars(declared, supplied)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Value should be "" (empty string), not the default
	if got["style"] != "" {
		t.Errorf("expected empty string value, got %q", got["style"])
	}
}


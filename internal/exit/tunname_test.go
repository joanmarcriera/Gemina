package exit

import (
	"errors"
	"testing"
)

func TestValidateTUNName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		ok    bool
	}{
		{"default", "gemina0", true},
		{"max length", "abcdefghijklmno", true},
		{"dash and underscore", "gem-ina_1", true},
		{"empty", "", false},
		{"too long", "abcdefghijklmnop", false},
		{"dot", ".", false},
		{"dotdot", "..", false},
		{"slash", "gem/0", false},
		{"colon alias", "gem:0", false},
		{"kernel template", "gem%d", false},
		{"space", "gem 0", false},
		{"tab", "gem\t0", false},
		{"nul", "gem\x000", false},
		{"non-ascii", "gemé", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTUNName(tt.input)
			if tt.ok && err != nil {
				t.Fatalf("ValidateTUNName(%q) = %v, want nil", tt.input, err)
			}
			if !tt.ok && !errors.Is(err, ErrInvalidTUNName) {
				t.Fatalf("ValidateTUNName(%q) = %v, want ErrInvalidTUNName", tt.input, err)
			}
		})
	}
}

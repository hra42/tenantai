package service

import (
	"strings"
	"testing"
)

func TestValidateServiceID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		// Valid IDs
		{name: "single char", id: "a", wantErr: false},
		{name: "simple alpha", id: "abc", wantErr: false},
		{name: "hyphenated", id: "my-service", wantErr: false},
		{name: "alphanumeric with hyphens", id: "a1-b2", wantErr: false},
		{name: "numeric only", id: "123", wantErr: false},
		{name: "mixed alpha-num segments", id: "svc-42-prod", wantErr: false},

		// Invalid IDs
		{name: "empty", id: "", wantErr: true},
		{name: "uppercase", id: "UPPER", wantErr: true},
		{name: "mixed case", id: "MyService", wantErr: true},
		{name: "space", id: "has space", wantErr: true},
		{name: "underscore", id: "bad_underscore", wantErr: true},
		{name: "leading hyphen", id: "-leading", wantErr: true},
		{name: "trailing hyphen", id: "trailing-", wantErr: true},
		{name: "double hyphen", id: "bad--id", wantErr: true},
		{name: "over 63 chars", id: strings.Repeat("a", 64), wantErr: true},
		{name: "exactly 63 chars", id: strings.Repeat("a", 63), wantErr: false},
		{name: "special chars", id: "svc@name", wantErr: true},
		{name: "dot", id: "svc.name", wantErr: true},
		{name: "slash", id: "svc/name", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServiceID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

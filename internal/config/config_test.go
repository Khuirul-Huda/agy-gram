package config

import (
	"testing"
)

func TestParseAllowedUserIDs(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty string",
			raw:       "",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "single user",
			raw:       "123456789",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "multiple users with spaces",
			raw:       " 123456789 , 987654321 , 11223344 ",
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "invalid user id",
			raw:       "12345,invalid_id",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAllowedUserIDs(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAllowedUserIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantCount {
				t.Errorf("parseAllowedUserIDs() got count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestIsUserAllowed(t *testing.T) {
	cfg := &Config{
		AllowedUserIDs: map[int64]struct{}{
			123456: {},
		},
	}

	if !cfg.IsUserAllowed(123456) {
		t.Errorf("expected user 123456 to be allowed")
	}

	if cfg.IsUserAllowed(999999) {
		t.Errorf("expected user 999999 to be rejected")
	}
}

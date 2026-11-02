package store

import (
	"errors"
	"strings"
	"testing"
)

// invalidateKey breaks k so that validate rejects it. Registry tests use this
// helper to exercise error propagation without duplicating validation rules.
func invalidateKey(k Key) Key {
	k.Environment = ""
	return k
}

func TestKeyValidate(t *testing.T) {
	tests := []struct {
		name      string
		key       Key
		wantErr   bool
		wantField string
	}{
		{
			name: "valid ascii",
			key: Key{
				Environment: "production",
				Source:      "api",
				Description: "main key",
			},
		},
		{
			name: "valid empty description",
			key: Key{
				Environment: "production",
				Source:      "api",
			},
		},
		{
			name: "valid utf8 description",
			key: Key{
				Environment: "production",
				Source:      "api",
				Description: "Pré-produção 🔑\nsecond line",
			},
		},
		{
			name: "valid identifier boundaries",
			key: Key{
				Environment: "!~",
				Source:      "AZaz09-._",
			},
		},
		{
			name: "valid max lengths",
			key: Key{
				Environment: strings.Repeat("e", maxIdentifierLen),
				Source:      strings.Repeat("s", maxIdentifierLen),
				Description: strings.Repeat("é", maxDescriptionLen/2),
			},
		},
		{
			name:      "empty environment",
			key:       Key{Source: "api"},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name:      "empty source",
			key:       Key{Environment: "production"},
			wantErr:   true,
			wantField: "source",
		},
		{
			name:      "environment with space",
			key:       Key{Environment: "prod env", Source: "api"},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name:      "source with space",
			key:       Key{Environment: "production", Source: "api v1"},
			wantErr:   true,
			wantField: "source",
		},
		{
			name:      "environment with tab",
			key:       Key{Environment: "prod\t", Source: "api"},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name:      "source with newline",
			key:       Key{Environment: "production", Source: "api\n"},
			wantErr:   true,
			wantField: "source",
		},
		{
			name:      "environment with carriage return",
			key:       Key{Environment: "pro\rduction", Source: "api"},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name:      "environment with nul byte",
			key:       Key{Environment: "prod\x00", Source: "api"},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name:      "environment with delete character",
			key:       Key{Environment: "prod\x7f", Source: "api"},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name:      "environment with non-ascii text",
			key:       Key{Environment: "pröd", Source: "api"},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name:      "source with non-ascii text",
			key:       Key{Environment: "production", Source: "apié"},
			wantErr:   true,
			wantField: "source",
		},
		{
			name: "environment too long",
			key: Key{
				Environment: strings.Repeat("e", maxIdentifierLen+1),
				Source:      "api",
			},
			wantErr:   true,
			wantField: "environment",
		},
		{
			name: "source too long",
			key: Key{
				Environment: "production",
				Source:      strings.Repeat("s", maxIdentifierLen+1),
			},
			wantErr:   true,
			wantField: "source",
		},
		{
			name: "description too long",
			key: Key{
				Environment: "production",
				Source:      "api",
				Description: strings.Repeat("a", maxDescriptionLen+1),
			},
			wantErr:   true,
			wantField: "description",
		},
		{
			name: "description with invalid utf8",
			key: Key{
				Environment: "production",
				Source:      "api",
				Description: string([]byte{0xff, 0xfe}),
			},
			wantErr:   true,
			wantField: "description",
		},
	}

	for _, tt := range tests {
		err := tt.key.validate()

		if !tt.wantErr {
			if err != nil {
				t.Errorf("%s: validate() returned an unexpected error: %v", tt.name, err)
			}
			continue
		}

		if err == nil {
			t.Errorf("%s: validate() returned nil, want an error", tt.name)
			continue
		}
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("%s: validate() error = %v, want ErrInvalidKey", tt.name, err)
		}
		if !strings.Contains(err.Error(), tt.wantField) {
			t.Errorf(
				"%s: validate() error = %q, want it to mention %q",
				tt.name,
				err,
				tt.wantField,
			)
		}
	}
}

package store

import (
	"strings"
	"testing"
)

// invalidateKey breaks k so that Validate rejects it. Registry tests use this
// helper to exercise error propagation without duplicating validation rules.
func invalidateKey(k Key) Key {
	k.Environment = ""
	return k
}

func TestKeyValidate(t *testing.T) {
	tests := []struct {
		name string
		key  Key
		// wantErr reports whether Validate must reject the key.
		wantErr bool
		// wantFields lists the fields whose names must appear in the error.
		wantFields []string
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
			name: "valid identifier characters",
			key: Key{
				Environment: "a-z.0_9",
				Source:      "az09-._",
			},
		},
		{
			name: "values at the length limit are allowed",
			key: Key{
				Environment: strings.Repeat("e", maxIdentifierLen),
				Source:      strings.Repeat("s", maxIdentifierLen),
				Description: strings.Repeat("d", maxDescriptionLen),
			},
		},
		{
			name:       "environment over the length limit",
			key:        Key{Environment: strings.Repeat("e", maxIdentifierLen+1), Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "source over the length limit",
			key:        Key{Environment: "production", Source: strings.Repeat("s", maxIdentifierLen+1)},
			wantErr:    true,
			wantFields: []string{"source"},
		},
		{
			name:       "description over the length limit",
			key:        Key{Environment: "production", Source: "api", Description: strings.Repeat("d", maxDescriptionLen+1)},
			wantErr:    true,
			wantFields: []string{"description"},
		},
		{
			name:       "empty environment",
			key:        Key{Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "empty source",
			key:        Key{Environment: "production"},
			wantErr:    true,
			wantFields: []string{"source"},
		},
		{
			name:       "empty environment and source",
			key:        Key{},
			wantErr:    true,
			wantFields: []string{"environment", "source"},
		},
		{
			name:       "environment with space",
			key:        Key{Environment: "prod env", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "source with space",
			key:        Key{Environment: "production", Source: "api v1"},
			wantErr:    true,
			wantFields: []string{"source"},
		},
		{
			name:       "environment with tab",
			key:        Key{Environment: "prod\t", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "source with newline",
			key:        Key{Environment: "production", Source: "api\n"},
			wantErr:    true,
			wantFields: []string{"source"},
		},
		{
			name:       "environment with carriage return",
			key:        Key{Environment: "pro\rduction", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "environment with nul byte",
			key:        Key{Environment: "prod\x00", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "environment with delete character",
			key:        Key{Environment: "prod\x7f", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "environment with non-ascii text",
			key:        Key{Environment: "pröd", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "source with non-ascii text",
			key:        Key{Environment: "production", Source: "apié"},
			wantErr:    true,
			wantFields: []string{"source"},
		},
		{
			name:       "environment with uppercase letter",
			key:        Key{Environment: "Production", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "source with uppercase letter",
			key:        Key{Environment: "production", Source: "API"},
			wantErr:    true,
			wantFields: []string{"source"},
		},
		{
			name:       "environment with disallowed symbol",
			key:        Key{Environment: "prod:env", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name:       "source with disallowed symbol",
			key:        Key{Environment: "production", Source: "api/v1"},
			wantErr:    true,
			wantFields: []string{"source"},
		},
		{
			name:       "environment with legacy visible ascii symbols",
			key:        Key{Environment: "!~", Source: "api"},
			wantErr:    true,
			wantFields: []string{"environment"},
		},
		{
			name: "description with invalid utf8",
			key: Key{
				Environment: "production",
				Source:      "api",
				Description: string([]byte{0xff, 0xfe}),
			},
			wantErr:    true,
			wantFields: []string{"description"},
		},
	}

	for _, tt := range tests {
		err := tt.key.Validate()

		if !tt.wantErr {
			if err != nil {
				t.Errorf("%s: Validate() returned an unexpected error: %v", tt.name, err)
			}
			continue
		}

		if err == nil {
			t.Errorf("%s: Validate() returned nil, want an error", tt.name)
			continue
		}
		for _, field := range tt.wantFields {
			if !strings.Contains(err.Error(), field) {
				t.Errorf(
					"%s: Validate() error = %q, want it to mention %q",
					tt.name,
					err,
					field,
				)
			}
		}
	}
}

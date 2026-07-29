package main

import "testing"

func TestValidateLoopbackDatabaseURLRejectsNonLocalOrIncompleteTargets(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "local PostgreSQL URL",
			raw:  "postgres://fixture:secret@127.0.0.1:5432/tidewise_local?sslmode=disable",
		},
		{
			name:    "remote host",
			raw:     "postgres://fixture:secret@database.example/tidewise",
			wantErr: true,
		},
		{
			name:    "missing database",
			raw:     "postgres://fixture:secret@localhost:5432",
			wantErr: true,
		},
		{
			name:    "non PostgreSQL scheme",
			raw:     "https://localhost/tidewise",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateLoopbackDatabaseURL(test.raw)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateLoopbackDatabaseURL(%q) error = %v", test.raw, err)
			}
		})
	}
}

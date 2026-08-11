package main

import "testing"

func TestValidateLocalDatabaseURLRejectsUnapprovedOrIncompleteTargets(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "external local PostgreSQL URL",
			raw:  "postgres://fixture:secret@host.docker.internal:5432/tidewise_local?sslmode=disable",
		},
		{
			name:    "remote host",
			raw:     "postgres://fixture:secret@database.example/tidewise",
			wantErr: true,
		},
		{
			name:    "missing database",
			raw:     "postgres://fixture:secret@host.docker.internal:5432",
			wantErr: true,
		},
		{
			name:    "non PostgreSQL scheme",
			raw:     "https://host.docker.internal/tidewise",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateLocalDatabaseURL(test.raw)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateLocalDatabaseURL(%q) error = %v", test.raw, err)
			}
		})
	}
}

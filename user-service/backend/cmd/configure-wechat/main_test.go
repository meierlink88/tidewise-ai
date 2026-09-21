package main

import (
	"errors"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/configuration"
	"strings"
	"testing"
)

func TestInvalidInputNeverUsesDatabase(t *testing.T) {
	t.Setenv("USER_DATABASE_URL", "")
	t.Setenv("USER_DATABASE_NAME", "")
	for _, input := range []string{`null`, `{}`, `{"app_id":"a","app_secret":""}`, `{"app_id":"a","app_secret":"b","extra":true}`, `{"app_id":"a","app_secret":"b"} {}`, strings.Repeat(" ", 4097)} {
		if err := run(strings.NewReader(input)); !errors.Is(err, biz.ErrUnavailable) {
			t.Fatal("invalid input accepted")
		}
	}
}

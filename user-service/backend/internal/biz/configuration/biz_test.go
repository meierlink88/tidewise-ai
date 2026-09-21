package configuration

import (
	"strings"
	"testing"
	"time"
)

func TestValidateConfiguration(t *testing.T) {
	for _, c := range []Credentials{{}, {AppID: "app"}, {AppID: "app", AppSecret: " secret"}, {AppID: "app\n", AppSecret: "secret"}, {AppID: "app", AppSecret: strings.Repeat("s", 257)}} {
		if _, err := NewEntry(c, time.Now()); err != ErrUnavailable {
			t.Fatal("invalid configuration accepted")
		}
	}
	e, err := NewEntry(Credentials{AppID: "app", AppSecret: "secret"}, time.Now())
	if err != nil || e.ID == "" {
		t.Fatal("valid configuration rejected")
	}
}

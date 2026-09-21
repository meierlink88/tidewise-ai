package conf

import (
	"testing"
)

func TestConfigurationSeparationAndTarget(t *testing.T) {
	env := map[string]string{"USER_DATABASE_URL": "postgres://user:password@localhost/tidewise_user_test", "USER_DATABASE_NAME": "tidewise_user_test"}
	get := func(k string) string { return env[k] }
	if _, err := load("", false, get); err != nil {
		t.Fatal(err)
	}
	if _, err := load("", true, get); err == nil {
		t.Fatal("server accepted missing secrets")
	}
	env["USER_SERVICE_TOKEN"] = "01234567890123456789012345678901"
	env["WECHAT_APP_ID"] = "app"
	env["WECHAT_APP_SECRET"] = "secret"
	c, err := load("", true, get)
	if err != nil || c.SessionTTLHours != 168 {
		t.Fatalf("defaults: %+v %v", c, err)
	}
	env["USER_DATABASE_URL"] += "?dbname=tidewise_local"
	if _, err = load("", false, get); err == nil {
		t.Fatal("accepted database query override")
	}
	env["USER_DATABASE_NAME"] = "tidewise_local"
	if _, err = load("", false, get); err == nil {
		t.Fatal("accepted Data database")
	}
}

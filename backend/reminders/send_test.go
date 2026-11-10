package reminders_test

import (
	"os"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/selfhostit/troye/migrations"
	"github.com/selfhostit/troye/reminders"
)

var template string

func TestMain(m *testing.M) {
	empty, err := os.MkdirTemp("", "troye-reminders-")
	if err != nil {
		panic(err)
	}
	migrated, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: empty})
	if err != nil {
		panic(err)
	}
	template = migrated.DataDir()
	if err := migrated.ResetBootstrapState(); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(empty)
	_ = os.RemoveAll(template)
	os.Exit(code)
}

func save(t testing.TB, a core.App, collection string, data map[string]any) *core.Record {
	t.Helper()
	c, err := a.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(data)
	if err := a.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func person(t testing.TB, a core.App, email string, profile map[string]any) string {
	users, _ := a.FindCollectionByNameOrId("users")
	u := core.NewRecord(users)
	u.SetEmail(email)
	u.SetPassword("password123")
	u.Set("role", "member")
	if err := a.Save(u); err != nil {
		t.Fatal(err)
	}
	profile["user"] = u.Id
	profile["onboarded"] = true
	save(t, a, "profiles", profile)
	return u.Id
}

func sent(t testing.TB, a core.App, user string) int {
	n, err := a.CountRecords(reminders.Collection, dbx.HashExp{"user": user})
	if err != nil {
		t.Fatal(err)
	}
	return int(n)
}

func TestSendEvening(t *testing.T) {
	a, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: template})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Cleanup()

	now := at("21:00") // a Monday
	empty := person(t, a, "empty@example.com", map[string]any{})
	full := person(t, a, "full@example.com", map[string]any{})
	for _, meal := range []string{"breakfast", "lunch", "dinner"} {
		save(t, a, "entries", map[string]any{"user": full, "date": "2026-11-09", "meal": meal, "name": meal, "kcal": 500})
	}
	off := person(t, a, "off@example.com", map[string]any{"reminders_off": []string{"evening"}})
	fasting := person(t, a, "fasting@example.com", map[string]any{"fasting_days": []string{"mon"}})

	for range 2 {
		if err := reminders.Send(a, reminders.Evening, now); err != nil {
			t.Fatal(err)
		}
	}
	// once, however often the job runs
	if n := sent(t, a, empty); n != 1 {
		t.Errorf("empty day: %d sent", n)
	}
	for name, user := range map[string]string{"full day": full, "turned off": off, "fasting day": fasting} {
		if n := sent(t, a, user); n != 0 {
			t.Errorf("%s: %d sent", name, n)
		}
	}
}

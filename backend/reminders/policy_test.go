package reminders_test

import (
	"testing"
	"time"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/reminders"
)

func at(hhmm string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", "2026-11-09 "+hhmm, time.Local)
	return t
}

func logged(meals ...string) daily.Day {
	return daily.Day{Date: "2026-11-09", Entries: len(meals), Meals: meals}
}

func TestEvening(t *testing.T) {
	awake := reminders.WindowFor("none", "")
	cases := []struct {
		name  string
		p     reminders.Person
		now   string
		title string
	}{
		{"nothing logged", reminders.Person{Window: awake}, "21:00", "Nothing logged today"},
		{"dinner missing", reminders.Person{Today: logged("breakfast", "lunch", "snacks"), Window: awake}, "21:00", "Dinner isn't logged yet"},
		{"two missing", reminders.Person{Today: logged("breakfast"), Window: awake}, "21:30", "Lunch and dinner aren't logged yet"},
		{"all three", reminders.Person{Today: logged("breakfast", "lunch", "dinner"), Window: awake}, "21:00", ""},
		{"too early", reminders.Person{Window: awake}, "20:59", ""},
		{"already sent", reminders.Person{Window: awake, Sent: true}, "21:00", ""},
		{"turned off", reminders.Person{Window: awake, Off: []reminders.Kind{reminders.Evening}}, "21:00", ""},
		{"a fasting day", reminders.Person{Window: awake, FastingDay: true}, "21:00", ""},
		{"16:8 skips breakfast on purpose", reminders.Person{Today: logged("lunch"), Window: reminders.WindowFor("16_8", "12:00")}, "21:00", ""},
		{"16:8 with nothing logged", reminders.Person{Window: reminders.WindowFor("16_8", "12:00")}, "21:00", "Nothing logged today"},
	}
	for _, c := range cases {
		msg, ok := reminders.Due(reminders.Evening, c.p, at(c.now))
		if ok != (c.title != "") || msg.Title != c.title {
			t.Errorf("%s: %v %q", c.name, ok, msg.Title)
		}
	}
}

func TestGap(t *testing.T) {
	awake := reminders.WindowFor("none", "")
	cases := []struct {
		name string
		last string // HH:MM today, or "" for yesterday evening
		now  string
		due  bool
	}{
		{"breakfast at 8, nothing since", "08:00", "16:00", true},
		{"breakfast at 8, only 7 hours", "08:00", "15:00", false},
		{"nothing today counts from 8:00", "", "16:00", true},
		{"not first thing in the morning", "", "09:00", false},
		{"quiet after 22:00", "08:00", "22:30", false},
	}
	for _, c := range cases {
		last := at("20:00").AddDate(0, 0, -1)
		if c.last != "" {
			last = at(c.last)
		}
		p := reminders.Person{LastEntry: last, Window: awake}
		msg, ok := reminders.Due(reminders.Gap, p, at(c.now))
		if ok != c.due {
			t.Errorf("%s: %v %+v", c.name, ok, msg)
		}
	}

	msg, _ := reminders.Due(reminders.Gap, reminders.Person{LastEntry: at("08:00"), Window: awake}, at("17:00"))
	if msg.Body != "Nothing logged for 9 hours. Add what you ate while you remember." {
		t.Errorf("body %q", msg.Body)
	}
	// people who stopped logging weeks ago aren't chased
	if _, ok := reminders.Due(reminders.Gap, reminders.Person{LastEntry: at("08:00").AddDate(0, 0, -20), Window: awake}, at("17:00")); ok {
		t.Error("nudged someone who stopped")
	}
	if _, ok := reminders.Due(reminders.Gap, reminders.Person{Window: awake}, at("17:00")); ok {
		t.Error("nudged someone who never logged")
	}
	// a 16:8 window from noon closes at 20:00, so 8 quiet hours never fit after a 12:30 lunch
	if _, ok := reminders.Due(reminders.Gap, reminders.Person{LastEntry: at("12:30"), Window: reminders.WindowFor("16_8", "12:00")}, at("20:00")); ok {
		t.Error("nudged within a short eating window")
	}
}

func TestWindowFor(t *testing.T) {
	if w := reminders.WindowFor("none", "12:00"); w.Start != 8*time.Hour || w.End != 22*time.Hour || w.Fasting {
		t.Errorf("no fasting %+v", w)
	}
	if w := reminders.WindowFor("16_8", "11:30"); w.Start != 11*time.Hour+30*time.Minute || w.End != 19*time.Hour+30*time.Minute || !w.Fasting {
		t.Errorf("16:8 %+v", w)
	}
	if w := reminders.WindowFor("14_10", ""); w.Start != 12*time.Hour {
		t.Errorf("no start %+v", w)
	}
}

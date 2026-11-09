// Package reminders nudges people to keep their diary complete: once in the
// evening when a meal is missing, and once a day after 8 hours of eating
// time without anything logged. This file decides who is due, purely from a
// Person at a moment; send.go loads people, sends and remembers.
package reminders

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/push"
)

// Kind is a kind of reminder; people turn kinds off on their profile.
type Kind string

const (
	// Evening goes out at 21:00 when a meal is missing from today.
	Evening Kind = "evening"
	// Gap goes out after 8 hours of eating time with nothing logged.
	Gap Kind = "gap"
)

// Window is when someone eats, as times of day. End can pass midnight.
type Window struct {
	Start, End time.Duration
	// Fasting is true for an intermittent fasting schedule.
	Fasting bool
}

// awake is the window for people who don't fast.
var awake = Window{Start: 8 * time.Hour, End: 22 * time.Hour}

// fastingHours are the hours each fasting schedule fasts.
var fastingHours = map[string]int{"12_12": 12, "14_10": 14, "16_8": 16, "18_6": 18, "20_4": 20, "omad": 23}

// WindowFor is the eating window of a profile's fasting schedule, which opens
// at eatingStart (HH:MM, noon when empty).
func WindowFor(fasting, eatingStart string) Window {
	hours, ok := fastingHours[fasting]
	if !ok {
		return awake
	}
	start, err := time.Parse("15:04", eatingStart)
	open := 12 * time.Hour
	if err == nil {
		open = time.Duration(start.Hour())*time.Hour + time.Duration(start.Minute())*time.Minute
	}
	return Window{Start: open, End: open + time.Duration(24-hours)*time.Hour, Fasting: true}
}

// Person is what the reminders need to know about someone, now.
type Person struct {
	// Today is today's diary so far.
	Today daily.Day
	// LastEntry is when they last logged anything; zero if never.
	LastEntry time.Time
	Window    Window
	// FastingDay is true on a day of their schedule set aside for fasting.
	FastingDay bool
	Off        []Kind
	// Sent is true when this kind already went out today.
	Sent bool
}

const (
	// eveningHour is when the evening reminder is due, until midnight
	eveningHour = 21
	// gap is how long without logging before a nudge
	gap = 8 * time.Hour
	// people who haven't logged in this long have stopped; don't chase them
	active = 14 * 24 * time.Hour
)

// Due is the reminder of a kind for someone now, and false when none is due.
func Due(k Kind, p Person, now time.Time) (push.Message, bool) {
	if p.Sent || p.FastingDay || slices.Contains(p.Off, k) {
		return push.Message{}, false
	}
	switch k {
	case Evening:
		return evening(p, now)
	case Gap:
		return nudge(p, now)
	}
	return push.Message{}, false
}

var mainMeals = []string{"breakfast", "lunch", "dinner"}

func evening(p Person, now time.Time) (push.Message, bool) {
	if now.Hour() < eveningHour {
		return push.Message{}, false
	}
	msg := push.Message{Body: "Add it before bed, so your week stays accurate.", URL: "/", Tag: "reminder-evening"}
	if !p.Today.Logged() {
		msg.Title = "Nothing logged today"
		return msg, true
	}
	// a fasting schedule skips meals on purpose, so only an empty day counts
	if p.Window.Fasting {
		return push.Message{}, false
	}
	var missing []string
	for _, m := range mainMeals {
		if !slices.Contains(p.Today.Meals, m) {
			missing = append(missing, m)
		}
	}
	switch len(missing) {
	case 0:
		return push.Message{}, false
	case 1:
		msg.Title = capitalize(missing[0]) + " isn't logged yet"
	default:
		msg.Title = capitalize(strings.Join(missing[:len(missing)-1], ", ")+" and "+missing[len(missing)-1]) + " aren't logged yet"
	}
	return msg, true
}

func nudge(p Person, now time.Time) (push.Message, bool) {
	if p.LastEntry.IsZero() || now.Sub(p.LastEntry) > active {
		return push.Message{}, false
	}
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	open, closes := midnight.Add(p.Window.Start), midnight.Add(p.Window.End)
	if now.Before(open) || now.After(closes) {
		return push.Message{}, false
	}
	// the gap counts from the last entry, or from when eating time opened today
	from := p.LastEntry
	if from.Before(open) {
		from = open
	}
	quiet := now.Sub(from)
	if quiet < gap {
		return push.Message{}, false
	}
	return push.Message{
		Title: "Anything to log?",
		Body:  fmt.Sprintf("Nothing logged for %d hours. Add what you ate while you remember.", int(quiet.Hours())),
		URL:   "/",
		Tag:   "reminder-gap",
	}, true
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

//go:build dev

package main

import (
	"context"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A store that invents everything and keeps nothing.
//
// The twelve methods below are written out because the screen has to have
// something on it to be worth looking at. The rest are generated blanks: they
// answer empty, which is a real state this product draws on purpose, so a
// screen reached through one of them is still a screen and not a crash.
//
// Turns are held in memory for the life of the process, so pressing things and
// typing into the dock behave. Nothing survives a restart, which is the point.
type store struct{}

var (
	said  []squirrel.Turn
	kept  []squirrel.Item
	gone        = map[int64]bool{}
	nextI int64 = 900
)

func now() time.Time { return time.Now() }

func note(id int64, text string, kind squirrel.ItemKind) squirrel.Item {
	return squirrel.Item{
		ID: id, RawText: text, Kind: kind, State: squirrel.ItemOpen,
		ReceivedAt: now().Add(-time.Duration(id) * 37 * time.Minute),
	}
}

func (store) OpenItems(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	all := []squirrel.Item{}
	for _, it := range everything() {
		if it.Kind == squirrel.ItemNote {
			all = append(all, it)
		}
	}
	all = standing(all)
	if len(all) > limit {
		return all[:limit], true, nil
	}
	return all, false, nil
}

func (store) Tasks(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	all := []squirrel.Item{}
	for _, it := range everything() {
		if it.Kind == squirrel.ItemTask {
			all = append(all, it)
		}
	}
	all = standing(all)
	if len(all) > limit {
		return all[:limit], true, nil
	}
	return all, false, nil
}

func chores() []squirrel.Chore {
	return []squirrel.Chore{
		{ID: 1, Name: "water the plants", Every: 7 * 24 * time.Hour, EveryDays: 7,
			SinceDays: 8, Active: true, EverDone: true},
		{ID: 2, Name: "sort the recycling", Every: 7 * 24 * time.Hour, EveryDays: 7,
			SinceDays: 9, Active: true, EverDone: true},
		{ID: 3, Name: "descale the kettle", Every: 14 * 24 * time.Hour, EveryDays: 14,
			SinceDays: 3, Active: true, EverDone: true},
	}
}

func (store) ActiveChores(_ context.Context, _ int64) ([]squirrel.Chore, error) {
	return chores(), nil
}

func (store) DueChores(_ context.Context, _ int64, _ time.Time) ([]squirrel.Chore, error) {
	return chores()[:2], nil
}

func (store) Waiting(_ context.Context, _ int64, _ time.Time) (squirrel.Waiting, error) {
	return squirrel.Waiting{Pile: 3, Tasks: 2, Chores: 2, Agenda: 1}, nil
}

func (store) Upcoming(_ context.Context, _ int64, _ time.Time, _ int) ([]squirrel.Moment, error) {
	return []squirrel.Moment{{
		ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour),
		Travel: 20 * time.Minute, Ready: 10 * time.Minute,
	}}, nil
}

func (store) KeptItems(_ context.Context, _ int64, _ int) ([]squirrel.Item, bool, error) {
	return nil, false, nil
}

func (store) HeldItems(_ context.Context, _ int64, _ int) ([]squirrel.HeldItem, bool, error) {
	return []squirrel.HeldItem{{
		ID: 9, Text: "the referral letter", State: squirrel.ItemWaiting,
		Because: "the surgery", Kind: squirrel.ItemNote,
	}}, false, nil
}

func (store) LatestCheckin(_ context.Context, _ int64) (squirrel.Checkin, bool, error) {
	return squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}, true, nil
}

// The conversation, in memory. Room-scoped, like the real one.
func (store) AppendTurn(_ context.Context, _ int64, room string, t squirrel.Turn) (squirrel.Turn, error) {
	t.ID = int64(len(said) + 1)
	t.Room = room
	t.SaidAt = now()
	said = append(said, t)
	return t, nil
}

func inRoom(room string) []squirrel.Turn {
	out := []squirrel.Turn{}
	for _, t := range said {
		if t.Room == room {
			out = append(out, t)
		}
	}
	return out
}

func (store) RecentTurns(_ context.Context, _ int64, room string, limit int) ([]squirrel.Turn, bool, error) {
	all := inRoom(room)
	if len(all) > limit {
		return all[len(all)-limit:], true, nil
	}
	return all, false, nil
}

func (store) TurnsBefore(_ context.Context, _ int64, room string, before int64, _ int) ([]squirrel.Turn, bool, error) {
	out := []squirrel.Turn{}
	for _, t := range inRoom(room) {
		if t.ID < before {
			out = append(out, t)
		}
	}
	return out, false, nil
}

// ---- the rest, answering empty ------------------------------------------
//
// Generated from the Store interface. An empty answer is a state this product
// draws on purpose, so every screen reached through one of these is still a
// screen.

func (store) OpenItemsAfter(_ context.Context, _, _ int64, _ int) ([]squirrel.Item, bool, error) {
	return nil, false, nil
}
func (store) SearchItems(_ context.Context, _ int64, q string, _ int) ([]squirrel.Item, bool, error) {
	out := []squirrel.Item{}
	for _, it := range everything() {
		if strings.Contains(strings.ToLower(it.RawText), strings.ToLower(q)) {
			out = append(out, it)
		}
	}
	return out, false, nil
}
func (store) InsertItem(_ context.Context, it squirrel.Item) (bool, error) {
	keep(it)
	return true, nil
}

func (store) InsertItemReturningID(_ context.Context, it squirrel.Item) (int64, error) {
	return keep(it), nil
}

// keep is what makes the screen answer a press. The real store writes a row;
// this holds it for the life of the process, so a note typed into a blank strip
// is on the rack when the board is drawn again.
func keep(it squirrel.Item) int64 {
	nextI++
	it.ID = nextI
	if it.Kind == "" {
		it.Kind = squirrel.ItemNote
	}
	it.State = squirrel.ItemOpen
	it.ReceivedAt = now()
	kept = append([]squirrel.Item{it}, kept...)
	return it.ID
}

// standing is what has not been answered. A press on a strip marks it gone, and
// the rack it was in stops drawing it.
func standing(all []squirrel.Item) []squirrel.Item {
	out := make([]squirrel.Item, 0, len(all))
	for _, it := range all {
		if !gone[it.ID] {
			out = append(out, it)
		}
	}
	return out
}
func (store) ArchivedTasks(_ context.Context, _ int64, _ int) ([]squirrel.Item, bool, error) {
	return nil, false, nil
}
func (store) Knowing(_ context.Context, _ int64) ([]string, error) { return nil, nil }
func (store) ForgetKnowing(_ context.Context, _ int64) error       { return nil }
func (store) SetItemKind(_ context.Context, _, _ int64, _ squirrel.ItemKind) (bool, error) {
	return false, nil
}
func (store) RecordCheckin(_ context.Context, _ int64, _ squirrel.Mood, _ string, _ time.Time) error {
	return nil
}
func (store) CheckinsSince(_ context.Context, _ int64, _ time.Time) ([]squirrel.Checkin, error) {
	return nil, nil
}
func (store) HoldItem(_ context.Context, _, _ int64, _ squirrel.ItemState, _ string, _ time.Time) (bool, error) {
	return false, nil
}
func (store) Unhold(_ context.Context, _, _ int64, _ time.Time) (bool, error) { return false, nil }
func (store) GoneQuiet(_ context.Context, _ int64, _ time.Time) (squirrel.HeldItem, bool, error) {
	return squirrel.HeldItem{}, false, nil
}
func (store) StillHolding(_ context.Context, _, _ int64, _ time.Time) (bool, error) {
	return false, nil
}
func (store) PickNow(_ context.Context, _ int64, _ time.Time, _ bool) (squirrel.Offer, bool, error) {
	return squirrel.Offer{
		Kind: squirrel.OfferChore, RefID: 1, Text: "bins out", Because: "it is bin day",
	}, true, nil
}
func (store) Did(_ context.Context, _ int64, _ squirrel.Offer, _ time.Time) error { return nil }
func (store) Refuse(_ context.Context, _ int64, _ squirrel.OfferKind, _ int64, _ time.Time) error {
	return nil
}
func (store) NotThisOne(_ context.Context, _ int64, _ squirrel.OfferKind, _ int64, _ time.Time) error {
	return nil
}
func (store) RecordAnswer(_ context.Context, _ int64, _ squirrel.OfferKind, _ int64, _ squirrel.OfferAnswer, _ time.Time) error {
	return nil
}
func (store) SaveSubscription(_ context.Context, _ int64, _ squirrel.Subscription) error { return nil }
func (store) StartTimer(_ context.Context, _ int64, _ string, _ time.Duration, _ time.Time) (squirrel.Timer, error) {
	return squirrel.Timer{}, nil
}
func (store) CurrentTimer(_ context.Context, _ int64) (squirrel.Timer, bool, error) {
	return squirrel.Timer{Label: "the kitchen", Started: now(), Ends: now().Add(6*time.Minute + 12*time.Second)}, true, nil
}
func (store) StopTimer(_ context.Context, _ int64) error       { return nil }
func (store) ArmRamp(_ context.Context, _ int64, _ bool) error { return nil }
func (store) RampDue(_ context.Context, _ int64, _ time.Time) (squirrel.Timer, bool, error) {
	return squirrel.Timer{}, false, nil
}
func (store) RampSaid(_ context.Context, _ int64, _ time.Time) error { return nil }
func (store) HushRamp(_ context.Context, _ int64, _ time.Time) error { return nil }
func (store) ItemByID(_ context.Context, _, id int64) (squirrel.Item, bool, error) {
	for _, it := range everything() {
		if it.ID == id {
			return it, true, nil
		}
	}
	return squirrel.Item{}, false, nil
}

// everything is every row the screen invents plus everything it has been told,
// which is what a press has to be able to find before it can answer it.
func everything() []squirrel.Item {
	out := append([]squirrel.Item{}, kept...)
	out = append(out,
		note(1, "the letter from the council about the bins", squirrel.ItemNote),
		note(2, "ask about the boiler service", squirrel.ItemNote),
		note(3, "that book Sam mentioned", squirrel.ItemNote),
		note(4, "book the MOT", squirrel.ItemTask),
		note(5, "ring the vet back", squirrel.ItemTask),
	)
	return out
}
func (store) SetItemState(_ context.Context, id int64, _ squirrel.ItemState, _ time.Time) error {
	gone[id] = true
	return nil
}

func (store) MoveItemState(_ context.Context, id int64, _, _ squirrel.ItemState, _ time.Time) (bool, error) {
	gone[id] = true
	return true, nil
}
func (store) LandedBadlyLatest(_ context.Context, _ int64, _ time.Time) (bool, error) {
	return false, nil
}
func (store) Reword(_ context.Context, _, _ int64, _ string) (bool, error) { return false, nil }
func (store) PromoteItem(_ context.Context, _, _ int64, _ time.Duration) (squirrel.Chore, bool, error) {
	return squirrel.Chore{}, false, nil
}
func (store) SearchChores(_ context.Context, _ int64, _ string, _ int) ([]squirrel.Chore, error) {
	return nil, nil
}
func (store) UpsertChore(_ context.Context, _ int64, _ string, _, _ time.Duration) (squirrel.Chore, error) {
	return squirrel.Chore{}, nil
}
func (store) UpsertChoreAsking(_ context.Context, _ int64, _ string, _, _ time.Duration, _ squirrel.Asking) (squirrel.Chore, error) {
	return squirrel.Chore{}, nil
}
func (store) SetChoreRhythm(_ context.Context, _, _ int64, _ time.Weekday, _ int) error   { return nil }
func (store) DeactivateChore(_ context.Context, _ int64) error                            { return nil }
func (store) RecordCompletion(_ context.Context, _, _ int64, _ string, _ time.Time) error { return nil }
func (store) CreateMoment(_ context.Context, _ int64, _ squirrel.Moment) (squirrel.Moment, error) {
	return squirrel.Moment{}, nil
}
func (store) MomentByID(_ context.Context, _, _ int64) (squirrel.Moment, bool, error) {
	return squirrel.Moment{}, false, nil
}
func (store) NotesFor(_ context.Context, _, _ int64) ([]squirrel.Item, error) { return nil, nil }
func (store) AttachNote(_ context.Context, _, _, _ int64) (bool, error)       { return false, nil }
func (store) DetachNote(_ context.Context, _, _ int64) (bool, error)          { return false, nil }
func (store) MarkRun(_ context.Context, _ int64, _ string, _ time.Time) error { return nil }
func (store) RunFor(_ context.Context, _ int64, _ time.Time) (squirrel.Run, bool, error) {
	return squirrel.Run{}, false, nil
}
func (store) EndRun(_ context.Context, _ int64) error                                    { return nil }
func (store) SaveSteps(_ context.Context, _ int64, _ *int64, _ string, _ []string) error { return nil }
func (store) NextStep(_ context.Context, _ int64) (squirrel.Step, bool, error) {
	return squirrel.Step{}, false, nil
}
func (store) StepDone(_ context.Context, _, _ int64, _ time.Time) error { return nil }
func (store) ClearSteps(_ context.Context, _ int64) error               { return nil }

// Who the dev screen is talking to. Invented, like everything else here, and
// deliberately without a picture: the monogram is the state a person who has
// never signed in through Authentik is actually in, so it is the one worth
// having on screen while the design is being looked at.
func (store) WhoIs(context.Context, int64) (squirrel.Whom, error) {
	return squirrel.Whom{Name: "Ronald Lokers", Handle: "ronald"}, nil
}

func (store) PersonFace(context.Context, int64) ([]byte, string, bool, error) {
	return nil, "", false, nil
}

func (store) EverythingSaid(_ context.Context, _ int64, limit int) ([]squirrel.Turn, bool, error) {
	if len(said) > limit {
		return said[len(said)-limit:], true, nil
	}
	return said, false, nil
}

func (store) EverythingBefore(_ context.Context, _, before int64, _ int) ([]squirrel.Turn, bool, error) {
	out := []squirrel.Turn{}
	for _, t := range said {
		if t.ID < before {
			out = append(out, t)
		}
	}
	return out, false, nil
}

func (store) MomentDone(_ context.Context, _, _ int64, _ time.Time) error { return nil }

func (store) TriagedSince(_ context.Context, _ int64, _ time.Time) ([]squirrel.Item, error) {
	return nil, nil
}

func (store) Notifying(_ context.Context, _ int64) (bool, error) { return false, nil }

func (store) StopNotifying(_ context.Context, _ int64, _ time.Time) error { return nil }

func (store) WhatWasSaid(_ context.Context, _ int64, _ int) ([]squirrel.Said, error) {
	return []squirrel.Said{
		{ID: 3, Title: "time to leave", Body: "the dentist is at 14:30", At: now().Add(-40 * time.Minute)},
		{ID: 2, Title: "the bins", Body: "they go out today", At: now().Add(-5 * time.Hour)},
		{ID: 1, Title: "how are you doing?", Body: "tap to say", At: now().Add(-9 * time.Hour)},
	}, nil
}

var (
	asked       []squirrel.Noticed
	nextAskedID int64 = 800
)

func (store) WhatWasNoticed(_ context.Context, _ int64) ([]squirrel.Noticed, error) {
	out := []squirrel.Noticed{
		{ID: 1, Kind: "note", RefID: 1, Words: "The code you need for this is in the note about the boiler."},
	}
	return append(out, asked...), nil
}

func (store) NotUseful(_ context.Context, _, id int64, _ time.Time) (bool, error) {
	for i, one := range asked {
		if one.ID == id {
			asked = append(asked[:i], asked[i+1:]...)
			return true, nil
		}
	}
	return true, nil
}

func (store) Notice(_ context.Context, _ int64, kind string, refID int64, words string, at time.Time) error {
	for i, one := range asked {
		if one.Kind == kind && one.RefID == refID {
			asked[i].Words, asked[i].At = words, at
			return nil
		}
	}
	nextAskedID++
	asked = append(asked, squirrel.Noticed{ID: nextAskedID, Kind: kind, RefID: refID, Words: words, At: at})
	return nil
}

package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCycle(t *testing.T) {
	c := defaultConfig()
	c.LongEvery = 2
	c.addTask("write report")
	var tm Timer
	tm.Reset(&c)
	now := time.Date(2026, 11, 21, 9, 0, 0, 0, time.Local)

	tm.Start(now)
	if got := fmtClock(tm.Left(now.Add(time.Second))); got != "24:59" {
		t.Fatalf("clock %s", got)
	}
	tm.Pause(now.Add(10 * time.Minute))
	if tm.Left(now.Add(time.Hour)) != 15*time.Minute {
		t.Fatal("pause should freeze remaining time")
	}
	if p := tm.Advance(&c, true, now); p != PhaseShort {
		t.Fatalf("want short break, got %v", p)
	}
	if tm.Remaining != 5*time.Minute {
		t.Fatal("break duration")
	}
	tm.Advance(&c, true, now)
	if p := tm.Advance(&c, true, now); p != PhaseLong {
		t.Fatalf("want long break after 2 sessions, got %v", p)
	}
	if c.TodayCount != 2 || c.TodayMin != 50 || c.Tasks[0].Pomos != 2 {
		t.Fatalf("stats %+v", c)
	}
	// skipping focus does not count
	tm.Advance(&c, true, now)
	tm.Advance(&c, false, now)
	if c.TodayCount != 2 {
		t.Fatal("skip counted")
	}
	// next day resets stats
	tm.Phase = PhaseFocus
	tm.Advance(&c, true, now.Add(24*time.Hour))
	if c.TodayCount != 1 {
		t.Fatalf("day roll: %d", c.TodayCount)
	}
}

func TestTasksAndPersist(t *testing.T) {
	c := defaultConfig()
	c.addTask("a")
	c.addTask("b")
	if c.currentTask().Text != "a" {
		t.Fatal("first task should be current")
	}
	c.toggleDone(0)
	if c.currentTask().Text != "b" {
		t.Fatal("current should move to next open task")
	}
	c.clearDone()
	if len(c.Tasks) != 1 {
		t.Fatal("clearDone")
	}
	p := filepath.Join(t.TempDir(), "x", "config.json")
	c.FocusMin = 50
	if err := saveConfig(p, &c); err != nil {
		t.Fatal(err)
	}
	l := loadConfig(p)
	if l.FocusMin != 50 || len(l.Tasks) != 1 || l.Tasks[0].Text != "b" {
		t.Fatalf("roundtrip %+v", l)
	}
}

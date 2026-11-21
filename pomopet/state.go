package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Phase int

const (
	PhaseFocus Phase = iota
	PhaseShort
	PhaseLong
)

func (p Phase) Label() string {
	switch p {
	case PhaseShort:
		return "BREAK"
	case PhaseLong:
		return "LONG BREAK"
	}
	return "FOCUS"
}

type Task struct {
	ID    int64  `json:"id"`
	Text  string `json:"text"`
	Done  bool   `json:"done"`
	Pomos int    `json:"pomos"`
}

type Config struct {
	FocusMin    int    `json:"focusMin"`
	ShortMin    int    `json:"shortMin"`
	LongMin     int    `json:"longMin"`
	LongEvery   int    `json:"longEvery"`
	AutoBreak   bool   `json:"autoBreak"`
	AutoFocus   bool   `json:"autoFocus"`
	Sound       bool   `json:"sound"`
	OnTop       bool   `json:"onTop"`
	PetX        int    `json:"petX"`
	PetY        int    `json:"petY"`
	HasPos      bool   `json:"hasPos"`
	Tasks       []Task `json:"tasks"`
	CurrentTask int64  `json:"currentTask"`
	StatsDate   string `json:"statsDate"`
	TodayCount  int    `json:"todayCount"`
	TodayMin    int    `json:"todayMin"`
}

func defaultConfig() Config {
	return Config{FocusMin: 25, ShortMin: 5, LongMin: 15, LongEvery: 4,
		AutoBreak: true, AutoFocus: false, Sound: true, OnTop: true}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (c *Config) sanitize() {
	d := defaultConfig()
	if c.FocusMin <= 0 {
		c.FocusMin = d.FocusMin
	}
	if c.ShortMin <= 0 {
		c.ShortMin = d.ShortMin
	}
	if c.LongMin <= 0 {
		c.LongMin = d.LongMin
	}
	if c.LongEvery <= 0 {
		c.LongEvery = d.LongEvery
	}
	c.FocusMin = clamp(c.FocusMin, 1, 600)
	c.ShortMin = clamp(c.ShortMin, 1, 600)
	c.LongMin = clamp(c.LongMin, 1, 600)
	c.LongEvery = clamp(c.LongEvery, 1, 12)
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "PomoPet", "config.json")
}

func loadConfig(path string) Config {
	c := defaultConfig()
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	c.sanitize()
	return c
}

func saveConfig(path string, c *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---- tasks ----

func (c *Config) addTask(text string) {
	id := time.Now().UnixNano()
	c.Tasks = append(c.Tasks, Task{ID: id, Text: text})
	if c.currentTask() == nil {
		c.CurrentTask = id
	}
}

func (c *Config) currentTask() *Task {
	for i := range c.Tasks {
		if c.Tasks[i].ID == c.CurrentTask && !c.Tasks[i].Done {
			return &c.Tasks[i]
		}
	}
	return nil
}

func (c *Config) toggleDone(i int) {
	if i < 0 || i >= len(c.Tasks) {
		return
	}
	t := &c.Tasks[i]
	t.Done = !t.Done
	if t.Done && t.ID == c.CurrentTask {
		c.CurrentTask = 0
		for _, o := range c.Tasks { // move on to the next open task
			if !o.Done {
				c.CurrentTask = o.ID
				break
			}
		}
	}
}

func (c *Config) deleteTask(i int) {
	if i < 0 || i >= len(c.Tasks) {
		return
	}
	c.Tasks = append(c.Tasks[:i], c.Tasks[i+1:]...)
}

func (c *Config) clearDone() {
	out := c.Tasks[:0]
	for _, t := range c.Tasks {
		if !t.Done {
			out = append(out, t)
		}
	}
	c.Tasks = out
}

func (t Task) ListText(current bool) string {
	mark := "○  "
	if t.Done {
		mark = "✓  "
	} else if current {
		mark = "▶  "
	}
	s := mark + t.Text
	if t.Pomos > 0 {
		plural := "s"
		if t.Pomos == 1 {
			plural = ""
		}
		s += fmt.Sprintf("   (%d pomodoro%s)", t.Pomos, plural)
	}
	return s
}

// ---- timer ----

type Timer struct {
	Phase     Phase
	Running   bool
	End       time.Time
	Remaining time.Duration // valid when !Running
	Cycle     int           // focus sessions completed in the current long-break cycle
}

func (c *Config) dur(p Phase) time.Duration {
	switch p {
	case PhaseShort:
		return time.Duration(c.ShortMin) * time.Minute
	case PhaseLong:
		return time.Duration(c.LongMin) * time.Minute
	}
	return time.Duration(c.FocusMin) * time.Minute
}

func (t *Timer) Left(now time.Time) time.Duration {
	if t.Running {
		d := t.End.Sub(now)
		if d < 0 {
			return 0
		}
		return d
	}
	return t.Remaining
}

func (t *Timer) Start(now time.Time) {
	if !t.Running {
		t.End = now.Add(t.Remaining)
		t.Running = true
	}
}

func (t *Timer) Pause(now time.Time) {
	if t.Running {
		t.Remaining = t.Left(now)
		t.Running = false
	}
}

func (t *Timer) Reset(c *Config) {
	t.Running = false
	t.Remaining = c.dur(t.Phase)
}

// Advance moves to the next phase. completed=false means the user skipped.
// It returns the new phase; the timer is left stopped at the full duration.
func (t *Timer) Advance(c *Config, completed bool, now time.Time) Phase {
	if t.Phase == PhaseFocus {
		next := PhaseShort
		if completed {
			c.rollStats(now)
			c.TodayCount++
			c.TodayMin += c.FocusMin
			if ct := c.currentTask(); ct != nil {
				ct.Pomos++
			}
			t.Cycle++
			if t.Cycle >= c.LongEvery {
				next = PhaseLong
				t.Cycle = 0
			}
		}
		t.Phase = next
	} else {
		t.Phase = PhaseFocus
	}
	t.Reset(c)
	return t.Phase
}

func (c *Config) rollStats(now time.Time) {
	d := now.Format("2006-01-02")
	if c.StatsDate != d {
		c.StatsDate = d
		c.TodayCount = 0
		c.TodayMin = 0
	}
}

func fmtClock(d time.Duration) string {
	s := int((d + time.Second - 1) / time.Second) // round up so 24:59.4 shows 25:00
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

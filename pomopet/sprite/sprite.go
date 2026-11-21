// Package sprite draws the pixel-art pet. It only needs a "fill rectangle"
// primitive, so the same code renders on Windows GDI and into PNGs (icon/preview).
package sprite

import "math"

type State int

const (
	Idle   State = iota // waiting to start
	Focus               // focus session running: typing on a laptop
	Break               // break running: sipping coffee
	Paused              // paused: dozing off
	Alert               // a session just ended: jumping and waving
)

// FillFunc fills an axis-aligned rectangle with a 0xRRGGBB colour.
type FillFunc func(x, y, w, h int, rgb uint32)

const (
	BodyW = 16 // grid cells occupied by body + arms (centre on this)
	GridW = 18 // full width including the coffee cup
	GridH = 8
)

const (
	cOrange = 0xD97757
	cDark   = 0x2A211C
	cBlush  = 0xF0A090
	cLaptop = 0x3B3F46
	cBase   = 0x9AA0A6
	cWhite  = 0xFFFFFF
	cSteam  = 0xB8B8B8
	cZ      = 0x6B7280
)

// Draw renders the pet with its top-left grid origin at (x0, y0), p pixels per
// grid cell, in state st, t seconds into the animation.
func Draw(fill FillFunc, x0, y0 int, p float64, st State, t float64) {
	r := func(gx, gy, gw, gh float64, c uint32) {
		x1 := x0 + int(math.Round(gx*p))
		y1 := y0 + int(math.Round(gy*p))
		x2 := x0 + int(math.Round((gx+gw)*p))
		y2 := y0 + int(math.Round((gy+gh)*p))
		if x2 > x1 && y2 > y1 {
			fill(x1, y1, x2-x1, y2-y1, c)
		}
	}
	frame := int(t / 0.15)
	dy := 0.0
	grounded := true
	switch st {
	case Idle:
		if int(t/0.7)%2 == 1 {
			dy = 0.25
		}
	case Break:
		if int(t/0.9)%2 == 1 {
			dy = 0.25
		}
	case Paused:
		dy = 0.4
		if int(t/1.2)%2 == 1 {
			dy = 0.6
		}
	case Alert:
		dy = -math.Abs(math.Sin(t*6)) * 1.5
		grounded = false
	}

	// legs
	for _, lx := range []float64{3, 5, 10, 12} {
		if grounded {
			r(lx, 6+dy, 1, 2-dy, cOrange)
		} else {
			r(lx, 6+dy, 1, 2, cOrange)
		}
	}
	// body
	r(2, dy, 12, 6, cOrange)

	// arms
	la, ra := 0.0, 0.0
	switch st {
	case Focus:
		if frame%2 == 0 {
			la = 0.5
		} else {
			ra = 0.5
		}
	case Alert:
		la, ra = -1.5, -1.0
		if frame%2 == 1 {
			la, ra = -1.0, -1.5
		}
	case Break:
		ra = -0.5
	case Paused:
		la, ra = 0.5, 0.5
	}
	r(0, 2+dy+la, 2, 2, cOrange)
	r(14, 2+dy+ra, 2, 2, cOrange)

	// eyes
	for _, ex := range []float64{4.5, 10.5} {
		switch st {
		case Idle:
			look := []float64{0, -0.5, 0, 0.5}[int(t/3)%4]
			if frame%25 == 0 {
				r(ex+look, 2.5+dy, 1, 0.5, cDark) // blink
			} else {
				r(ex+look, 1.5+dy, 1, 1.5, cDark)
			}
		case Focus:
			if frame%30 == 0 {
				r(ex, 3+dy, 1, 0.5, cDark)
			} else {
				r(ex, 2.5+dy, 1, 1, cDark) // looking down at the laptop
			}
		case Break: // happy ^ ^
			r(ex-0.25, 2.5+dy, 0.5, 0.5, cDark)
			r(ex+0.25, 2+dy, 0.5, 0.5, cDark)
			r(ex+0.75, 2.5+dy, 0.5, 0.5, cDark)
		case Paused: // closed
			r(ex-0.25, 2.75+dy, 1.5, 0.4, cDark)
		case Alert: // wide open
			r(ex, 1+dy, 1, 2, cDark)
		}
	}
	if st == Break {
		r(3, 3.5+dy, 1, 0.5, cBlush)
		r(12, 3.5+dy, 1, 0.5, cBlush)
	}
	if st == Alert {
		r(7.5, 4+dy, 1, 1, cDark) // "o" mouth
	}

	switch st {
	case Focus: // laptop
		r(4, 3.5, 8, 2.6, cLaptop)
		r(7.5, 4.4, 1, 0.6, 0xE8E8E8)
		r(3, 6.1, 10, 0.5, cBase)
	case Break: // coffee cup held in the right arm
		cy := 0.6 + dy
		r(15.6, cy, 1.9, 2.1, cDark)
		r(15.85, cy+0.25, 1.4, 1.6, cWhite)
		r(15.85, cy+0.25, 1.4, 0.4, 0x8B5A3C)
		r(17.5, cy+0.6, 0.45, 0.9, cDark)
		s := float64(frame % 4)
		r(16.1+0.3*math.Mod(s, 2), cy-0.7-0.15*s, 0.4, 0.4, cSteam)
		r(16.8-0.3*math.Mod(s, 2), cy-1.4-0.15*s, 0.4, 0.4, cSteam)
	case Paused: // z z
		ph := math.Mod(t/0.8, 3)
		drawZ(r, 14.5+ph*0.3, -0.5-ph*0.5, 0.3)
		drawZ(r, 16.2+ph*0.2, -2.2-ph*0.4, 0.22)
	}
}

func drawZ(r func(gx, gy, gw, gh float64, c uint32), x, y, u float64) {
	r(x, y, 3*u, u, cZ)
	r(x+2*u, y+u, u, u, cZ)
	r(x+u, y+2*u, u, u, cZ)
	r(x, y+3*u, 3*u, u, cZ)
}

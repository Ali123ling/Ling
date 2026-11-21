package main

import (
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"pomopet/sprite"
)

const appName = "PomoPet"

// Pet window layout in 96-DPI units.
const (
	petW     = 200
	petH     = 146
	petCell  = 7.0
	petTop   = 84
	bubL     = 8
	bubT     = 4
	bubR     = 192
	bubB     = 62
	keyColor = 0xFF00FF // transparent colour key
)

// timers / menu ids / control ids
const (
	tmrTick  = 1
	tmrClick = 2

	cmdToggle = 1001 + iota
	cmdReset
	cmdSkip
	cmdPanel
	cmdOnTop
	cmdAutostart
	cmdShowPet
	cmdExit

	idFocus     = 101
	idShort     = 102
	idLong      = 103
	idEvery     = 104
	idAutoBreak = 105
	idAutoFocus = 106
	idSound     = 107
	idApply     = 108
	idTaskEdit  = 201
	idAdd       = 202
	idList      = 203
	idCurrent   = 204
	idDone      = 205
	idDelete    = 206
	idClearDone = 207
	idStats     = 301
)

var (
	hInst    uintptr
	petHwnd  uintptr
	panHwnd  uintptr
	appIcon  uintptr
	scale    = 1.0
	cfg      Config
	cfgPath  string
	tm       Timer
	appStart = time.Now()

	alertMsg   string
	alertUntil time.Time

	dragging  bool
	dragMoved bool
	dragPt    POINT
	dragWin   RECT

	brushes            = map[uint32]uintptr{}
	fTime, fLabel, fTx uintptr
	uiFont             uintptr
	ctl                = map[int]uintptr{}
	lastTip            string
	lastStats          string
)

func sc(v int) int           { return int(math.Round(float64(v) * scale)) }
func scu(v int) uintptr      { return uintptr(sc(v)) }
func now() time.Time         { return time.Now() }
func saveCfg()               { _ = saveConfig(cfgPath, &cfg) }
func invalidate(h uintptr)   { pInvalidateRect.Call(h, 0, 0) }
func visible(h uintptr) bool { r, _, _ := pIsWindowVisible.Call(h); return r != 0 }

func brush(c uint32) uintptr {
	if b, ok := brushes[c]; ok {
		return b
	}
	b, _, _ := pCreateSolidBrush.Call(rgb(c))
	brushes[c] = b
	return b
}

func fill(dc uintptr, x, y, w, h int, c uint32) {
	r := RECT{int32(x), int32(y), int32(x + w), int32(y + h)}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), brush(c))
}

func mkFont(px, weight int) uintptr {
	f, _, _ := pCreateFontW.Call(uintptr(-sc(px)), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, up("Segoe UI"))
	return f
}

func main() {
	runtime.LockOSThread()

	// single instance
	_, _, e := pCreateMutexW.Call(0, 0, up("Local\\PomoPetSingleInstance"))
	if e == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		pMessageBoxW.Call(0, up("PomoPet is already running — look for the little orange buddy on your desktop or in the tray."), up(appName), MB_ICONINFO)
		return
	}

	pSetProcessDPIAware.Call()
	dc, _, _ := pGetDC.Call(0)
	dpi, _, _ := pGetDeviceCaps.Call(dc, LOGPIXELSX)
	pReleaseDC.Call(0, dc)
	if dpi > 0 {
		scale = float64(dpi) / 96.0
	}

	hInst, _, _ = pGetModuleHandleW.Call(0)
	appIcon, _, _ = pLoadIconW.Call(hInst, up("APP"))
	if appIcon == 0 {
		appIcon, _, _ = pLoadIconW.Call(0, IDI_APP)
	}

	cfgPath = configPath()
	cfg = loadConfig(cfgPath)
	cfg.rollStats(now())
	tm.Reset(&cfg)

	fTime = mkFont(24, 700)
	fLabel = mkFont(11, 700)
	fTx = mkFont(12, 400)
	uiFont = mkFont(14, 400)

	createPet()
	createPanel()
	trayAdd()

	if len(cfg.Tasks) == 0 && !cfg.HasPos {
		showAlert("Hi! Double-click me for tasks & settings", 8*time.Second)
	}

	var msg MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if panHwnd != 0 && visible(panHwnd) {
			if r, _, _ := pIsDialogMessageW.Call(panHwnd, uintptr(unsafe.Pointer(&msg))); r != 0 {
				continue
			}
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func registerClass(name string, proc uintptr, style uint32, bg uintptr) {
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc := WNDCLASSEXW{
		Style: style, WndProc: proc, Instance: hInst, Icon: appIcon, IconSm: appIcon,
		Cursor: cur, Background: bg, ClassName: u16(name),
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
}

// ======================= pet window =======================

func createPet() {
	registerClass("PomoPetWnd", syscall.NewCallback(petProc), CS_DBLCLKS, 0)
	x, y := cfg.PetX, cfg.PetY
	if !cfg.HasPos || !onScreen(x, y) {
		var wa RECT
		pSystemParametersInfoW.Call(SPI_GETWORKAREA, 0, uintptr(unsafe.Pointer(&wa)), 0)
		x = int(wa.R) - sc(petW) - sc(24)
		y = int(wa.B) - sc(petH) - sc(8)
	}
	ex := uintptr(WS_EX_LAYERED | WS_EX_TOOLWINDOW)
	if cfg.OnTop {
		ex |= WS_EX_TOPMOST
	}
	petHwnd, _, _ = pCreateWindowExW.Call(ex, up("PomoPetWnd"), up(appName), WS_POPUP,
		uintptr(x), uintptr(y), scu(petW), scu(petH), 0, 0, hInst, 0)
	pSetLayeredWindowAttributes.Call(petHwnd, rgb(keyColor), 0, LWA_COLORKEY)
	pShowWindow.Call(petHwnd, SW_SHOW)
	pSetTimer.Call(petHwnd, tmrTick, 100, 0)
}

func onScreen(x, y int) bool {
	vx, _, _ := pGetSystemMetrics.Call(SM_XVIRTUAL)
	vy, _, _ := pGetSystemMetrics.Call(SM_YVIRTUAL)
	vw, _, _ := pGetSystemMetrics.Call(SM_CXVIRTUAL)
	vh, _, _ := pGetSystemMetrics.Call(SM_CYVIRTUAL)
	l, t := int(int32(vx)), int(int32(vy))
	return x > l-sc(petW)/2 && y > t-sc(20) && x < l+int(vw)-sc(petW)/2 && y < t+int(vh)-sc(petH)/2
}

func petState(n time.Time) sprite.State {
	switch {
	case n.Before(alertUntil):
		return sprite.Alert
	case tm.Running && tm.Phase == PhaseFocus:
		return sprite.Focus
	case tm.Running:
		return sprite.Break
	case tm.Remaining < cfg.dur(tm.Phase):
		return sprite.Paused
	}
	return sprite.Idle
}

func petProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		paintPet(hwnd)
		return 0
	case WM_TIMER:
		if wp == tmrClick {
			pKillTimer.Call(hwnd, tmrClick)
			toggle()
			return 0
		}
		onTick()
		return 0
	case WM_LBUTTONDOWN:
		dragging, dragMoved = true, false
		dragPt = cursorPos()
		dragWin = windowRect(hwnd)
		pSetCapture.Call(hwnd)
		return 0
	case WM_MOUSEMOVE:
		if dragging {
			p := cursorPos()
			dx, dy := int(p.X-dragPt.X), int(p.Y-dragPt.Y)
			if abs(dx) > 3 || abs(dy) > 3 {
				dragMoved = true
			}
			if dragMoved {
				pSetWindowPos.Call(hwnd, 0, uintptr(int(dragWin.L)+dx), uintptr(int(dragWin.T)+dy), 0, 0,
					SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
			}
		}
		return 0
	case WM_LBUTTONUP:
		if dragging {
			dragging = false
			pReleaseCapture.Call()
			if dragMoved {
				r := windowRect(hwnd)
				cfg.PetX, cfg.PetY, cfg.HasPos = int(r.L), int(r.T), true
				saveCfg()
			} else {
				dt, _, _ := pGetDoubleClickTime.Call()
				pSetTimer.Call(hwnd, tmrClick, dt, 0)
			}
		}
		return 0
	case WM_LBUTTONDBLCLK:
		pKillTimer.Call(hwnd, tmrClick)
		showPanel()
		return 0
	case WM_CAPTURECHANGED:
		dragging = false
		return 0
	case WM_RBUTTONUP:
		showMenu(hwnd, false)
		return 0
	case WM_TRAY:
		switch uint32(lp) {
		case WM_LBUTTONUP:
			showPet(true)
		case WM_LBUTTONDBLCLK:
			showPanel()
		case WM_RBUTTONUP:
			showMenu(hwnd, true)
		}
		return 0
	case WM_QUERYENDSESSION:
		saveCfg()
		return 1
	case WM_DESTROY:
		trayDelete()
		saveCfg()
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func onTick() {
	n := now()
	if tm.Running && tm.Left(n) <= 0 {
		finishPhase(true)
	}
	if visible(petHwnd) {
		invalidate(petHwnd)
	}
	trayTip()
	if panHwnd != 0 && visible(panHwnd) {
		updateStats()
	}
}

func toggle() {
	n := now()
	alertUntil = time.Time{}
	if tm.Running {
		tm.Pause(n)
	} else {
		tm.Start(n)
	}
	invalidate(petHwnd)
}

func finishPhase(completed bool) {
	prev := tm.Phase
	next := tm.Advance(&cfg, completed, now())
	saveCfg()
	refreshList()
	if !completed {
		alertUntil = time.Time{}
		invalidate(petHwnd)
		return
	}
	var title, body string
	if prev == PhaseFocus {
		mins := cfg.ShortMin
		if next == PhaseLong {
			mins = cfg.LongMin
		}
		title = "Focus session complete!"
		body = "Time for a " + strconv.Itoa(mins) + "-minute break. Stand up, stretch, drink some water."
		showAlert("Break time! Step away for "+strconv.Itoa(mins)+" min!", 12*time.Second)
		if cfg.AutoBreak {
			tm.Start(now())
		}
	} else {
		title = "Break is over"
		body = "Ready for another " + strconv.Itoa(cfg.FocusMin) + "-minute focus session?"
		if cfg.AutoFocus {
			showAlert("Back to it! Focus started", 10*time.Second)
			tm.Start(now())
		} else {
			showAlert("Break over — click me to focus", 12*time.Second)
		}
	}
	showPet(true)
	if cfg.Sound {
		pMessageBeep.Call(MB_ICONASTERISK)
	}
	trayBalloon(title, body)
}

func showAlert(msg string, d time.Duration) {
	alertMsg = msg
	alertUntil = now().Add(d)
	invalidate(petHwnd)
}

func showPet(show bool) {
	if show {
		pShowWindow.Call(petHwnd, SW_SHOW)
		z := HWND_TOPMOST
		if !cfg.OnTop {
			z = 0
		}
		pSetWindowPos.Call(petHwnd, z, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
	} else {
		pShowWindow.Call(petHwnd, SW_HIDE)
	}
}

func paintPet(hwnd uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	w, h := sc(petW), sc(petH)
	mdc, _, _ := pCreateCompatibleDC.Call(hdc)
	bmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	oldBmp, _, _ := pSelectObject.Call(mdc, bmp)

	fill(mdc, 0, 0, w, h, keyColor)
	n := now()
	drawBubble(mdc, n)
	t := n.Sub(appStart).Seconds()
	p := petCell * scale
	x0 := (w - int(sprite.BodyW*p)) / 2
	sprite.Draw(func(x, y, ww, hh int, c uint32) { fill(mdc, x, y, ww, hh, c) }, x0, sc(petTop), p, petState(n), t)

	pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), mdc, 0, 0, SRCCOPY)
	pSelectObject.Call(mdc, oldBmp)
	pDeleteObject.Call(bmp)
	pDeleteDC.Call(mdc)
	pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

func drawText(dc uintptr, s string, font uintptr, color uint32, l, t, r, b int, flags uint32) {
	old, _, _ := pSelectObject.Call(dc, font)
	pSetTextColor.Call(dc, rgb(color))
	rc := RECT{int32(sc(l)), int32(sc(t)), int32(sc(r)), int32(sc(b))}
	ws, _ := syscall.UTF16FromString(s)
	pDrawTextW.Call(dc, uintptr(unsafe.Pointer(&ws[0])), uintptr(len(ws)-1), uintptr(unsafe.Pointer(&rc)),
		uintptr(flags|DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS))
	pSelectObject.Call(dc, old)
}

func drawBubble(dc uintptr, n time.Time) {
	const ink = 0x2B2622
	pen, _, _ := pCreatePen.Call(0, scu(2), rgb(ink))
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	oldBr, _, _ := pSelectObject.Call(dc, brush(0xFFFFFF))
	pRoundRect.Call(dc, scu(bubL), scu(bubT), scu(bubR), scu(bubB), scu(16), scu(16))
	cx := petW / 2
	pts := []POINT{{int32(sc(cx - 8)), int32(sc(bubB - 2))}, {int32(sc(cx + 8)), int32(sc(bubB - 2))}, {int32(sc(cx)), int32(sc(bubB + 9))}}
	pPolygon.Call(dc, uintptr(unsafe.Pointer(&pts[0])), 3)
	fill(dc, sc(cx-5), sc(bubB-5), sc(10), sc(5), 0xFFFFFF)
	pSelectObject.Call(dc, oldBr)
	pSelectObject.Call(dc, oldPen)
	pDeleteObject.Call(pen)
	pSetBkMode.Call(dc, TRANSPARENT)

	full := cfg.dur(tm.Phase)
	left := tm.Left(n)
	idle := !tm.Running && left >= full
	paused := !tm.Running && !idle

	accent := uint32(0xD97757)
	switch tm.Phase {
	case PhaseShort:
		accent = 0x3E9A62
	case PhaseLong:
		accent = 0x3B78C4
	}
	label := tm.Phase.Label()
	labelColor := accent
	if paused {
		label = "PAUSED"
		labelColor = 0x8A8F98
	}

	drawText(dc, fmtClock(left), fTime, ink, 20, 8, 110, 36, DT_LEFT)
	drawText(dc, label, fLabel, labelColor, 100, 10, 180, 24, DT_RIGHT)

	// cycle dots: sessions done toward the long break
	nd := cfg.LongEvery
	if nd > 8 {
		nd = 8
	}
	for i := 0; i < nd; i++ {
		x := 180 - (nd-i)*9 + 2
		c := uint32(0xE2DED9)
		if i < tm.Cycle {
			c = 0xD97757
		}
		fill(dc, sc(x), sc(27), sc(6), sc(6), c)
	}

	var line string
	lineColor := uint32(0x5A524C)
	switch {
	case n.Before(alertUntil):
		line, lineColor = alertMsg, accent
	case idle && tm.Phase != PhaseFocus:
		line = "Click me to start your break"
	case idle && cfg.currentTask() == nil:
		line = "Click: start · Right-click: menu"
	case cfg.currentTask() != nil && tm.Phase == PhaseFocus:
		line = "▶ " + cfg.currentTask().Text
	case tm.Phase != PhaseFocus:
		line = "Relax, stretch, look away ~"
	default:
		line = "Stay focused!"
	}
	drawText(dc, line, fTx, lineColor, 20, 36, 180, 50, DT_LEFT)

	// progress bar
	bx, bw := 20, 160
	fill(dc, sc(bx), sc(52), sc(bw), sc(4), 0xEEEAE6)
	if full > 0 {
		pw := int(float64(sc(bw)) * (1 - float64(left)/float64(full)))
		fill(dc, sc(bx), sc(52), pw, sc(4), accent)
	}
}

// ======================= menus / tray =======================

func showMenu(hwnd uintptr, fromTray bool) {
	m, _, _ := pCreatePopupMenu.Call()
	add := func(id int, text string, flags uintptr) { pAppendMenuW.Call(m, flags, uintptr(id), up(text)) }
	sep := func() { pAppendMenuW.Call(m, MF_SEPARATOR, 0, 0) }
	full := cfg.dur(tm.Phase)
	switch {
	case tm.Running:
		add(cmdToggle, "Pause", MF_STRING)
	case tm.Remaining < full:
		add(cmdToggle, "Resume", MF_STRING)
	default:
		add(cmdToggle, "Start "+strings.ToLower(tm.Phase.Label()), MF_STRING)
	}
	add(cmdReset, "Reset this session", MF_STRING)
	if tm.Phase == PhaseFocus {
		add(cmdSkip, "Skip to break", MF_STRING)
	} else {
		add(cmdSkip, "Skip break", MF_STRING)
	}
	sep()
	add(cmdPanel, "Tasks && settings…", MF_STRING)
	chk := func(b bool) uintptr {
		if b {
			return MF_CHECKED
		}
		return 0
	}
	if fromTray {
		add(cmdShowPet, "Show pet on desktop", chk(visible(petHwnd)))
	} else {
		add(cmdShowPet, "Hide pet (keep in tray)", MF_STRING)
	}
	add(cmdOnTop, "Always on top", chk(cfg.OnTop))
	add(cmdAutostart, "Start with Windows", chk(autostartEnabled()))
	sep()
	add(cmdExit, "Exit", MF_STRING)

	p := cursorPos()
	pSetForegroundWindow.Call(hwnd)
	id, _, _ := pTrackPopupMenu.Call(m, TPM_RETURNCMD|TPM_RIGHTBUTTON, uintptr(p.X), uintptr(p.Y), 0, hwnd, 0)
	pDestroyMenu.Call(m)
	pPostMessageW.Call(hwnd, 0, 0, 0) // per MSDN, so the menu closes properly from the tray

	switch int(id) {
	case cmdToggle:
		toggle()
	case cmdReset:
		tm.Reset(&cfg)
		alertUntil = time.Time{}
	case cmdSkip:
		finishPhase(false)
	case cmdPanel:
		showPanel()
	case cmdShowPet:
		showPet(!visible(petHwnd))
	case cmdOnTop:
		cfg.OnTop = !cfg.OnTop
		z := HWND_NOTOPMOST
		if cfg.OnTop {
			z = HWND_TOPMOST
		}
		pSetWindowPos.Call(petHwnd, z, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
		saveCfg()
	case cmdAutostart:
		setAutostart(!autostartEnabled())
	case cmdExit:
		saveCfg()
		pDestroyWindowCall(petHwnd)
	}
	invalidate(petHwnd)
}

var pDestroyWindow = user32.NewProc("DestroyWindow")

func pDestroyWindowCall(h uintptr) { pDestroyWindow.Call(h) }

func trayData() *NOTIFYICONDATAW {
	d := &NOTIFYICONDATAW{Hwnd: petHwnd, UID: 1}
	d.CbSize = uint32(unsafe.Sizeof(*d))
	return d
}

func trayAdd() {
	d := trayData()
	d.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	d.UCallbackMessage = WM_TRAY
	d.HIcon = appIcon
	copyU16(d.SzTip[:], appName)
	pShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(d)))
}

func trayDelete() { pShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(trayData()))) }

func trayTip() {
	n := now()
	s := appName + " — " + strings.ToLower(tm.Phase.Label()) + " " + fmtClock(tm.Left(n))
	if !tm.Running {
		s += " (stopped)"
	}
	if s == lastTip {
		return
	}
	lastTip = s
	d := trayData()
	d.UFlags = NIF_TIP
	copyU16(d.SzTip[:], s)
	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(d)))
}

func trayBalloon(title, body string) {
	d := trayData()
	d.UFlags = NIF_INFO
	d.DwInfoFlags = NIIF_INFO
	copyU16(d.SzInfoTitle[:], title)
	copyU16(d.SzInfo[:], body)
	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(d)))
}

// ======================= autostart (HKCU Run) =======================

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func openRun(access uintptr) (uintptr, bool) {
	var k uintptr
	r, _, _ := pRegOpenKeyExW.Call(HKEY_CURRENT_USER, up(runKey), 0, access, uintptr(unsafe.Pointer(&k)))
	return k, r == 0
}

func autostartEnabled() bool {
	k, ok := openRun(KEY_QUERY_VALUE)
	if !ok {
		return false
	}
	defer pRegCloseKey.Call(k)
	r, _, _ := pRegQueryValueExW.Call(k, up(appName), 0, 0, 0, 0)
	return r == 0
}

func setAutostart(on bool) {
	k, ok := openRun(KEY_SET_VALUE)
	if !ok {
		return
	}
	defer pRegCloseKey.Call(k)
	if !on {
		pRegDeleteValueW.Call(k, up(appName))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	v, _ := syscall.UTF16FromString(`"` + exe + `"`)
	pRegSetValueExW.Call(k, up(appName), 0, REG_SZ, uintptr(unsafe.Pointer(&v[0])), uintptr(len(v)*2))
}

// ======================= tasks & settings panel =======================

const panW, panH = 440, 520

func createPanel() {
	registerClass("PomoPetPanel", syscall.NewCallback(panelProc), 0, func() uintptr {
		b, _, _ := pGetStockObject.Call(WHITE_BRUSH)
		return b
	}())
	style := uintptr(WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX)
	rc := RECT{0, 0, int32(sc(panW)), int32(sc(panH))}
	pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&rc)), style, 0, WS_EX_CONTROLPARENT)
	panHwnd, _, _ = pCreateWindowExW.Call(WS_EX_CONTROLPARENT, up("PomoPetPanel"), up(appName+" — Tasks & Settings"), style,
		0x80000000, 0x80000000, uintptr(rc.R-rc.L), uintptr(rc.B-rc.T), 0, 0, hInst, 0) // CW_USEDEFAULT

	c := func(id int, class, text string, style uint32, ex uintptr, x, y, w, h int) uintptr {
		hw, _, _ := pCreateWindowExW.Call(ex, up(class), up(text), uintptr(WS_CHILD|WS_VISIBLE|style),
			scu(x), scu(y), scu(w), scu(h), panHwnd, uintptr(id), hInst, 0)
		sendMsg(hw, WM_SETFONT, uiFont, 1)
		if id > 0 {
			ctl[id] = hw
		}
		return hw
	}
	num := func(id, x, y int) {
		c(id, "EDIT", "", WS_TABSTOP|ES_NUMBER|ES_AUTOHSCROLL, WS_EX_CLIENTEDGE, x, y, 56, 24)
	}
	lbl := func(text string, x, y, w int) { c(0, "STATIC", text, 0, 0, x, y+3, w, 20) }

	c(0, "BUTTON", "Timer", BS_GROUPBOX, 0, 12, 6, 416, 156)
	lbl("Focus (minutes)", 28, 30, 120)
	num(idFocus, 150, 30)
	lbl("Short break (min)", 226, 30, 130)
	num(idShort, 356, 30)
	lbl("Long break (min)", 28, 62, 120)
	num(idLong, 150, 62)
	lbl("Long break every", 226, 62, 130)
	num(idEvery, 356, 62)
	c(idAutoBreak, "BUTTON", "Start break automatically when focus ends", BS_AUTOCHECKBOX|WS_TABSTOP, 0, 28, 94, 300, 22)
	c(idAutoFocus, "BUTTON", "Start next focus automatically after a break", BS_AUTOCHECKBOX|WS_TABSTOP, 0, 28, 118, 300, 22)
	c(idSound, "BUTTON", "Sound", BS_AUTOCHECKBOX|WS_TABSTOP, 0, 340, 94, 76, 22)
	c(idApply, "BUTTON", "Save", WS_TABSTOP, 0, 336, 120, 80, 28)

	c(0, "BUTTON", "To-do", BS_GROUPBOX, 0, 12, 170, 416, 304)
	te := c(idTaskEdit, "EDIT", "", WS_TABSTOP|ES_AUTOHSCROLL, WS_EX_CLIENTEDGE, 28, 194, 300, 26)
	sendMsg(te, EM_SETCUEBANNER, 1, up("What do you want to get done?"))
	c(idAdd, "BUTTON", "Add", WS_TABSTOP|BS_DEFPUSHBUTTON, 0, 336, 193, 80, 28)
	c(idList, "LISTBOX", "", WS_TABSTOP|WS_VSCROLL|LBS_NOTIFY|LBS_NOINTEGRALHEIGHT, WS_EX_CLIENTEDGE, 28, 228, 388, 196)
	c(idCurrent, "BUTTON", "Work on this", WS_TABSTOP, 0, 28, 432, 96, 28)
	c(idDone, "BUTTON", "Done / undo", WS_TABSTOP, 0, 128, 432, 92, 28)
	c(idDelete, "BUTTON", "Delete", WS_TABSTOP, 0, 224, 432, 92, 28)
	c(idClearDone, "BUTTON", "Clear done", WS_TABSTOP, 0, 320, 432, 96, 28)

	c(idStats, "STATIC", "", 0, 0, 14, 484, 412, 24)

	loadSettingsIntoPanel()
	refreshList()
}

func setChecked(id int, b bool) {
	v := uintptr(0)
	if b {
		v = 1
	}
	sendMsg(ctl[id], BM_SETCHECK, v, 0)
}

func checked(id int) bool { return sendMsg(ctl[id], BM_GETCHECK, 0, 0) == 1 }

func loadSettingsIntoPanel() {
	setText(ctl[idFocus], strconv.Itoa(cfg.FocusMin))
	setText(ctl[idShort], strconv.Itoa(cfg.ShortMin))
	setText(ctl[idLong], strconv.Itoa(cfg.LongMin))
	setText(ctl[idEvery], strconv.Itoa(cfg.LongEvery))
	setChecked(idAutoBreak, cfg.AutoBreak)
	setChecked(idAutoFocus, cfg.AutoFocus)
	setChecked(idSound, cfg.Sound)
}

func applySettings() {
	num := func(id, def int) int {
		v, err := strconv.Atoi(strings.TrimSpace(getText(ctl[id])))
		if err != nil {
			return def
		}
		return v
	}
	oldFull := cfg.dur(tm.Phase)
	cfg.FocusMin = num(idFocus, cfg.FocusMin)
	cfg.ShortMin = num(idShort, cfg.ShortMin)
	cfg.LongMin = num(idLong, cfg.LongMin)
	cfg.LongEvery = num(idEvery, cfg.LongEvery)
	cfg.AutoBreak = checked(idAutoBreak)
	cfg.AutoFocus = checked(idAutoFocus)
	cfg.Sound = checked(idSound)
	cfg.sanitize()
	// a session that hasn't started yet picks up the new length immediately
	if !tm.Running && tm.Remaining >= oldFull {
		tm.Reset(&cfg)
	}
	if tm.Cycle >= cfg.LongEvery {
		tm.Cycle = 0
	}
	saveCfg()
	loadSettingsIntoPanel()
	invalidate(petHwnd)
	showAlert("Settings saved!", 3*time.Second)
}

func refreshList() {
	if panHwnd == 0 {
		return
	}
	lb := ctl[idList]
	sel := int(int32(sendMsg(lb, LB_GETCURSEL, 0, 0)))
	sendMsg(lb, LB_RESETCONTENT, 0, 0)
	for _, t := range cfg.Tasks {
		sendMsg(lb, LB_ADDSTRING, 0, up(t.ListText(t.ID == cfg.CurrentTask)))
	}
	if sel >= len(cfg.Tasks) {
		sel = len(cfg.Tasks) - 1
	}
	if sel >= 0 {
		sendMsg(lb, LB_SETCURSEL, uintptr(sel), 0)
	}
	updateStats()
	invalidate(petHwnd)
}

func updateStats() {
	cfg.rollStats(now())
	open := 0
	for _, t := range cfg.Tasks {
		if !t.Done {
			open++
		}
	}
	s := "Today: " + strconv.Itoa(cfg.TodayCount) + " focus sessions (" + strconv.Itoa(cfg.TodayMin) +
		" min)   ·   " + strconv.Itoa(open) + " open tasks"
	if s != lastStats {
		lastStats = s
		setText(ctl[idStats], s)
	}
}

func selected() int { return int(int32(sendMsg(ctl[idList], LB_GETCURSEL, 0, 0))) }

func addTaskFromEdit() {
	s := strings.TrimSpace(getText(ctl[idTaskEdit]))
	if s == "" {
		return
	}
	cfg.addTask(s)
	setText(ctl[idTaskEdit], "")
	saveCfg()
	refreshList()
	sendMsg(ctl[idList], LB_SETCURSEL, uintptr(len(cfg.Tasks)-1), 0)
	pSetFocus.Call(ctl[idTaskEdit])
}

func showPanel() {
	loadSettingsIntoPanel()
	refreshList()
	pShowWindow.Call(panHwnd, SW_RESTORE)
	pShowWindow.Call(panHwnd, SW_SHOW)
	pSetForegroundWindow.Call(panHwnd)
	pSetFocus.Call(ctl[idTaskEdit])
}

func panelProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		id, code := int(wp&0xFFFF), int((wp>>16)&0xFFFF)
		switch id {
		case IDOK: // Enter key
			f, _, _ := pGetFocus.Call()
			if f == ctl[idTaskEdit] {
				addTaskFromEdit()
			} else if f == ctl[idList] {
				cfg.toggleDone(selected())
				saveCfg()
				refreshList()
			} else {
				applySettings()
			}
		case 2: // IDCANCEL (Esc)
			pShowWindow.Call(hwnd, SW_HIDE)
		case idApply:
			applySettings()
		case idAdd:
			addTaskFromEdit()
		case idList:
			if code == LBN_DBLCLK {
				cfg.toggleDone(selected())
				saveCfg()
				refreshList()
			}
		case idCurrent:
			if i := selected(); i >= 0 && i < len(cfg.Tasks) {
				cfg.Tasks[i].Done = false
				cfg.CurrentTask = cfg.Tasks[i].ID
				saveCfg()
				refreshList()
			}
		case idDone:
			cfg.toggleDone(selected())
			saveCfg()
			refreshList()
		case idDelete:
			cfg.deleteTask(selected())
			saveCfg()
			refreshList()
		case idClearDone:
			cfg.clearDone()
			saveCfg()
			refreshList()
		}
		return 0
	case WM_CTLCOLORSTATIC:
		pSetBkMode.Call(wp, TRANSPARENT)
		b, _, _ := pGetStockObject.Call(WHITE_BRUSH)
		return b
	case WM_CLOSE:
		pShowWindow.Call(hwnd, SW_HIDE)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

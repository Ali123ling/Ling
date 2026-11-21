package main

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	pRegisterClassExW           = user32.NewProc("RegisterClassExW")
	pCreateWindowExW            = user32.NewProc("CreateWindowExW")
	pDefWindowProcW             = user32.NewProc("DefWindowProcW")
	pGetMessageW                = user32.NewProc("GetMessageW")
	pTranslateMessage           = user32.NewProc("TranslateMessage")
	pDispatchMessageW           = user32.NewProc("DispatchMessageW")
	pPostQuitMessage            = user32.NewProc("PostQuitMessage")
	pShowWindow                 = user32.NewProc("ShowWindow")
	pInvalidateRect             = user32.NewProc("InvalidateRect")
	pBeginPaint                 = user32.NewProc("BeginPaint")
	pEndPaint                   = user32.NewProc("EndPaint")
	pSetTimer                   = user32.NewProc("SetTimer")
	pKillTimer                  = user32.NewProc("KillTimer")
	pSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	pLoadCursorW                = user32.NewProc("LoadCursorW")
	pLoadIconW                  = user32.NewProc("LoadIconW")
	pSetCapture                 = user32.NewProc("SetCapture")
	pReleaseCapture             = user32.NewProc("ReleaseCapture")
	pGetCursorPos               = user32.NewProc("GetCursorPos")
	pSetWindowPos               = user32.NewProc("SetWindowPos")
	pGetWindowRect              = user32.NewProc("GetWindowRect")
	pCreatePopupMenu            = user32.NewProc("CreatePopupMenu")
	pAppendMenuW                = user32.NewProc("AppendMenuW")
	pTrackPopupMenu             = user32.NewProc("TrackPopupMenu")
	pDestroyMenu                = user32.NewProc("DestroyMenu")
	pSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	pSendMessageW               = user32.NewProc("SendMessageW")
	pPostMessageW               = user32.NewProc("PostMessageW")
	pGetWindowTextW             = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	pSetWindowTextW             = user32.NewProc("SetWindowTextW")
	pIsDialogMessageW           = user32.NewProc("IsDialogMessageW")
	pGetFocus                   = user32.NewProc("GetFocus")
	pSetFocus                   = user32.NewProc("SetFocus")
	pFillRect                   = user32.NewProc("FillRect")
	pDrawTextW                  = user32.NewProc("DrawTextW")
	pGetDC                      = user32.NewProc("GetDC")
	pReleaseDC                  = user32.NewProc("ReleaseDC")
	pAdjustWindowRectEx         = user32.NewProc("AdjustWindowRectEx")
	pSystemParametersInfoW      = user32.NewProc("SystemParametersInfoW")
	pGetSystemMetrics           = user32.NewProc("GetSystemMetrics")
	pMessageBeep                = user32.NewProc("MessageBeep")
	pSetProcessDPIAware         = user32.NewProc("SetProcessDPIAware")
	pIsWindowVisible            = user32.NewProc("IsWindowVisible")
	pGetDoubleClickTime         = user32.NewProc("GetDoubleClickTime")
	pMessageBoxW                = user32.NewProc("MessageBoxW")

	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pPolygon                = gdi32.NewProc("Polygon")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")
	pGetStockObject         = gdi32.NewProc("GetStockObject")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW     = kernel32.NewProc("CreateMutexW")

	pShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")

	pRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	pRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	pRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	pRegCloseKey      = advapi32.NewProc("RegCloseKey")
)

const (
	WS_POPUP            = 0x80000000
	WS_CHILD            = 0x40000000
	WS_VISIBLE          = 0x10000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_MINIMIZEBOX      = 0x00020000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	WS_TABSTOP          = 0x00010000
	WS_GROUP            = 0x00020000
	WS_EX_LAYERED       = 0x00080000
	WS_EX_TOPMOST       = 0x00000008
	WS_EX_TOOLWINDOW    = 0x00000080
	WS_EX_CLIENTEDGE    = 0x00000200
	WS_EX_CONTROLPARENT = 0x00010000

	CS_DBLCLKS = 0x0008
	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001

	ES_AUTOHSCROLL       = 0x0080
	ES_NUMBER            = 0x2000
	BS_DEFPUSHBUTTON     = 0x0001
	BS_AUTOCHECKBOX      = 0x0003
	BS_GROUPBOX          = 0x0007
	LBS_NOTIFY           = 0x0001
	LBS_NOINTEGRALHEIGHT = 0x0100

	WM_CREATE          = 0x0001
	WM_DESTROY         = 0x0002
	WM_PAINT           = 0x000F
	WM_CLOSE           = 0x0010
	WM_ERASEBKGND      = 0x0014
	WM_SETFONT         = 0x0030
	WM_COMMAND         = 0x0111
	WM_TIMER           = 0x0113
	WM_CTLCOLORSTATIC  = 0x0138
	WM_MOUSEMOVE       = 0x0200
	WM_LBUTTONDOWN     = 0x0201
	WM_LBUTTONUP       = 0x0202
	WM_LBUTTONDBLCLK   = 0x0203
	WM_RBUTTONUP       = 0x0205
	WM_CAPTURECHANGED  = 0x0215
	WM_APP             = 0x8000
	WM_TRAY            = WM_APP + 1
	WM_QUERYENDSESSION = 0x0011

	BM_GETCHECK     = 0x00F0
	BM_SETCHECK     = 0x00F1
	LB_ADDSTRING    = 0x0180
	LB_RESETCONTENT = 0x0184
	LB_SETCURSEL    = 0x0186
	LB_GETCURSEL    = 0x0188
	LBN_DBLCLK      = 2
	EM_SETCUEBANNER = 0x1501

	SW_HIDE    = 0
	SW_SHOW    = 5
	SW_RESTORE = 9

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040

	LWA_COLORKEY = 0x1

	MF_STRING    = 0x0000
	MF_SEPARATOR = 0x0800
	MF_CHECKED   = 0x0008
	MF_GRAYED    = 0x0001

	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100

	NIM_ADD     = 0
	NIM_MODIFY  = 1
	NIM_DELETE  = 2
	NIF_MESSAGE = 0x01
	NIF_ICON    = 0x02
	NIF_TIP     = 0x04
	NIF_INFO    = 0x10
	NIIF_INFO   = 0x01

	DT_LEFT         = 0x0000
	DT_CENTER       = 0x0001
	DT_RIGHT        = 0x0002
	DT_VCENTER      = 0x0004
	DT_SINGLELINE   = 0x0020
	DT_NOPREFIX     = 0x0800
	DT_END_ELLIPSIS = 0x8000

	TRANSPARENT = 1
	SRCCOPY     = 0x00CC0020
	LOGPIXELSX  = 88

	SPI_GETWORKAREA = 0x0030
	SM_XVIRTUAL     = 76
	SM_YVIRTUAL     = 77
	SM_CXVIRTUAL    = 78
	SM_CYVIRTUAL    = 79

	IDC_ARROW = 32512
	IDC_HAND  = 32649
	IDI_APP   = 32512

	MB_ICONASTERISK = 0x40
	MB_OK           = 0
	MB_ICONINFO     = 0x40

	HKEY_CURRENT_USER = 0x80000001
	KEY_QUERY_VALUE   = 0x0001
	KEY_SET_VALUE     = 0x0002
	REG_SZ            = 1

	WHITE_BRUSH = 0
	IDOK        = 1
)

var HWND_TOPMOST = ^uintptr(0)       // -1
var HWND_NOTOPMOST = ^uintptr(0) - 1 // -2

type POINT struct{ X, Y int32 }
type RECT struct{ L, T, R, B int32 }

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	Private uint32
}

type WNDCLASSEXW struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type NOTIFYICONDATAW struct {
	CbSize           uint32
	Hwnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         GUID
	HBalloonIcon     uintptr
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func up(s string) uintptr { return uintptr(unsafe.Pointer(u16(s))) }

func rgb(c uint32) uintptr { // 0xRRGGBB -> COLORREF 0x00BBGGRR
	return uintptr((c>>16)&0xFF | (c & 0xFF00) | (c&0xFF)<<16)
}

func sendMsg(h uintptr, msg uint32, w, l uintptr) uintptr {
	r, _, _ := pSendMessageW.Call(h, uintptr(msg), w, l)
	return r
}

func getText(h uintptr) string {
	n, _, _ := pGetWindowTextLengthW.Call(h)
	buf := make([]uint16, n+1)
	pGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func setText(h uintptr, s string) { pSetWindowTextW.Call(h, up(s)) }

func copyU16(dst []uint16, s string) {
	src, _ := syscall.UTF16FromString(s)
	if len(src) > len(dst) {
		src = src[:len(dst)]
		src[len(src)-1] = 0
	}
	copy(dst, src)
}

func cursorPos() POINT {
	var p POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

func windowRect(h uintptr) RECT {
	var r RECT
	pGetWindowRect.Call(h, uintptr(unsafe.Pointer(&r)))
	return r
}

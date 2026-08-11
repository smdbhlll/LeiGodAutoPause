//go:build windows

package gui

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"leigod-auto-pause/internal/app"
	"leigod-auto-pause/internal/config"
	"leigod-auto-pause/internal/discovery"
	"leigod-auto-pause/internal/leigod"
	"leigod-auto-pause/internal/processes"
	"leigod-auto-pause/internal/startup"
)

const (
	windowClass = "LeiGodAutoPauseNativeWindow"
	windowTitle = "雷神自动暂停"

	wsVisible       = 0x10000000
	wsChild         = 0x40000000
	wsTabStop       = 0x00010000
	wsBorder        = 0x00800000
	wsVScroll       = 0x00200000
	wsCaption       = 0x00C00000
	wsSysMenu       = 0x00080000
	wsMinimizeBox   = 0x00020000
	wsClipChildren  = 0x02000000
	wsOverlapped    = 0x00000000
	wsExClientEdge  = 0x00000200
	bsPushButton    = 0x00000000
	bsFlat          = 0x00008000
	bsOwnerDraw     = 0x0000000B
	bsAutoCheckbox  = 0x00000003
	bsGroupBox      = 0x00000007
	esAutoHScroll   = 0x00000080
	esPassword      = 0x00000020
	esNumber        = 0x00002000
	cbsDropDownList = 0x00000003
	cbsOwnerDraw    = 0x00000010
	cbsHasStrings   = 0x00000200
	lbsNotify       = 0x00000001
	lbsNoIntegral   = 0x00000100
	lvsReport       = 0x00000001
	lvsSingleSel    = 0x00000004
	lvsShowSelect   = 0x00000008
	lvsNoHeader     = 0x00004000
	lvsNoSortHead   = 0x00008000
	lvexGridlines   = 0x00000001
	lvexFullRow     = 0x00000020
	lvexDoubleBuf   = 0x00010000

	wmCreate         = 0x0001
	wmNull           = 0x0000
	wmDestroy        = 0x0002
	wmClose          = 0x0010
	wmSettingChange  = 0x001A
	wmSetFont        = 0x0030
	wmCommand        = 0x0111
	wmDrawItem       = 0x002B
	wmNotify         = 0x004E
	wmTimer          = 0x0113
	wmSize           = 0x0005
	wmUser           = 0x0400
	wmTray           = wmUser + 10
	wmAppResult      = 0x8001
	wmCtlColorEdit   = 0x0133
	wmCtlColorList   = 0x0134
	wmCtlColorBtn    = 0x0135
	wmCtlColorStatic = 0x0138
	bnClicked        = 0
	gclpBackground   = ^uintptr(9) // GCLP_HBRBACKGROUND (-10)

	lbAddString     = 0x0180
	lbResetContent  = 0x0184
	lbGetCurSel     = 0x0188
	lbErr           = ^uintptr(0)
	lbnDoubleClick  = 2
	cbAddString     = 0x0143
	cbGetText       = 0x0148
	cbGetTextLength = 0x0149
	cbGetCurSel     = 0x0147
	cbSetCurSel     = 0x014E
	lvmFirst        = 0x1000
	lvmSetBkColor   = lvmFirst + 1
	lvmDeleteAll    = lvmFirst + 9
	lvmGetNext      = lvmFirst + 12
	lvmSetExtStyle  = lvmFirst + 54
	lvmSetTextColor = lvmFirst + 36
	lvmSetTextBk    = lvmFirst + 38
	lvmGetHeader    = lvmFirst + 31
	lvmInsertCol    = lvmFirst + 97
	lvmInsertItem   = lvmFirst + 77
	lvmSetItemText  = lvmFirst + 116
	lvniSelected    = 0x0002
	lvcfFmt         = 0x0001
	lvcfWidth       = 0x0002
	lvcfText        = 0x0004
	lvcfSubItem     = 0x0008
	lvifText        = 0x0001
	sizeMinimized   = 1
	nimAdd          = 0
	nimModify       = 1
	nimDelete       = 2
	nifMessage      = 1
	nifIcon         = 2
	nifTip          = 4
	nifInfo         = 0x10
	niifInfo        = 1
	niifError       = 3
	wmLButtonUp     = 0x0202
	wmLButtonDbl    = 0x0203
	wmRButtonUp     = 0x0205
	wmContextMenu   = 0x007B
	odsSelected     = 0x0001
	odsDisabled     = 0x0004
	cdsPrepaint     = 0x00000001
	cdsItemPrepaint = 0x00010001
	cdrfNewFont     = 0x00000002
	cdrfSkipDefault = 0x00000004
	cdrfNotifyItem  = 0x00000020
	nmCustomDraw    = -12
	mfString        = 0x0000
	mfSeparator     = 0x0800
	tpmRightButton  = 0x0002
	tpmReturnCmd    = 0x0100
	imageIcon       = 1
	lrLoadFromFile  = 0x0010
	lrDefaultSize   = 0x0040
	hdmFirst        = 0x1200
	hdmGetItem      = hdmFirst + 11
	hdiText         = 0x0002

	swHide          = 0
	swShow          = 5
	swRestore       = 9
	colorWindow     = 5
	idcArrow        = 32512
	idiApplication  = 32512
	swShowMinimized = 2
	mbOK            = 0x00000000
	mbIconError     = 0x00000010
	mbIconInfo      = 0x00000040
	mbIconWarning   = 0x00000030
	mbYesNo         = 0x00000004
	mbYesNoCancel   = 0x00000003
	idYes           = 6
	idNo            = 7
	errorAlreadyRun = 183

	viewOverview = 1
	viewGames    = 2
	viewSettings = 3
)

const (
	toggleTrackWidth      int32 = 46
	toggleTrackHeight     int32 = 22
	toggleKnobSize        int32 = 16
	toggleKnobTravel      int32 = 22
	toggleSupersample     int32 = 4
	toggleFrameMillis           = 8
	toggleAnimationMillis       = 200
	stretchModeHalftone         = 4
	rasterCopy                  = 0x00CC0020
)

const (
	idNavOverview = 1001 + iota
	idNavGames
	idNavSettings
	idPauseNow
	idOpenLeiGod
	idCheckAccount
	idGameList
	idScanRunning
	idScanInstalled
	idBrowseGame
	idAddManual
	idToggleGame
	idRemoveGame
	idCandidateList
	idCandidateAdd
	idCandidateBack
	idSaveSettings
	idClearToken
	idClearPassword
	idSettingsAccount
	idBrowseLeiGod
	idFindLeiGod
)

const (
	idTrayShow = 3001 + iota
	idTrayPause
	idTrayExit
)

//go:embed assets/app.ico
var appIconData []byte

type point struct{ X, Y int32 }

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type drawItem struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HWndItem   uintptr
	HDC        uintptr
	Rect       rect
	ItemData   uintptr
}

type notifyHeader struct {
	HWndFrom uintptr
	IDFrom   uintptr
	Code     int32
}

type baseCustomDraw struct {
	Header     notifyHeader
	DrawStage  uint32
	HDC        uintptr
	Rect       rect
	ItemSpec   uintptr
	ItemState  uint32
	ItemLParam uintptr
}

type customDraw struct {
	baseCustomDraw
	TextColor uint32
	TextBk    uint32
	Font      uintptr
	SubItem   uint32
}

type headerItem struct {
	Mask       uint32
	Width      int32
	Text       *uint16
	Bitmap     uintptr
	TextLength int32
	Format     int32
	LParam     uintptr
	Image      int32
	Order      int32
	Type       uint32
	Filter     uintptr
	State      uint32
}

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type wndClassEx struct {
	Size       uint32
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
	IconSmall  uintptr
}

type openFileName struct {
	Size            uint32
	Owner           uintptr
	Instance        uintptr
	Filter          *uint16
	CustomFilter    *uint16
	MaxCustomFilter uint32
	FilterIndex     uint32
	File            *uint16
	MaxFile         uint32
	FileTitle       *uint16
	MaxFileTitle    uint32
	InitialDir      *uint16
	Title           *uint16
	Flags           uint32
	FileOffset      uint16
	FileExtension   uint16
	DefaultExt      *uint16
	CustomData      uintptr
	Hook            uintptr
	TemplateName    *uint16
	Reserved        uintptr
	Reserved2       uint32
	FlagsEx         uint32
}

type initCommonControls struct {
	Size uint32
	ICC  uint32
}

type listColumn struct {
	Mask       uint32
	Fmt        int32
	Width      int32
	Text       *uint16
	TextMax    int32
	SubItem    int32
	Image      int32
	Order      int32
	MinWidth   int32
	DefaultWid int32
	IdealWidth int32
}

type listItem struct {
	Mask       uint32
	Item       int32
	SubItem    int32
	State      uint32
	StateMask  uint32
	Text       *uint16
	TextMax    int32
	Image      int32
	Param      uintptr
	Indent     int32
	GroupID    int32
	Columns    uint32
	ColumnPtr  *uint32
	ColumnFmt  *int32
	GroupIndex int32
}

type notifyIconData struct {
	Size        uint32
	Window      uintptr
	ID          uint32
	Flags       uint32
	Callback    uint32
	Icon        uintptr
	Tip         [128]uint16
	State       uint32
	StateMask   uint32
	Info        [256]uint16
	Version     uint32
	InfoTitle   [64]uint16
	InfoFlags   uint32
	GUID        [16]byte
	BalloonIcon uintptr
}

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	comdlg32               = syscall.NewLazyDLL("comdlg32.dll")
	comctl32               = syscall.NewLazyDLL("comctl32.dll")
	uxtheme                = syscall.NewLazyDLL("uxtheme.dll")
	shell32                = syscall.NewLazyDLL("shell32.dll")
	dwmapi                 = syscall.NewLazyDLL("dwmapi.dll")
	advapi32               = syscall.NewLazyDLL("advapi32.dll")
	registerClassEx        = user32.NewProc("RegisterClassExW")
	createWindowEx         = user32.NewProc("CreateWindowExW")
	defWindowProc          = user32.NewProc("DefWindowProcW")
	showWindow             = user32.NewProc("ShowWindow")
	updateWindow           = user32.NewProc("UpdateWindow")
	setForegroundWindow    = user32.NewProc("SetForegroundWindow")
	redrawWindow           = user32.NewProc("RedrawWindow")
	systemParametersInfo   = user32.NewProc("SystemParametersInfoW")
	setWindowRgn           = user32.NewProc("SetWindowRgn")
	setClassLongPtr        = user32.NewProc("SetClassLongPtrW")
	findWindow             = user32.NewProc("FindWindowW")
	destroyWindow          = user32.NewProc("DestroyWindow")
	postQuitMessage        = user32.NewProc("PostQuitMessage")
	getMessage             = user32.NewProc("GetMessageW")
	translateMessage       = user32.NewProc("TranslateMessage")
	dispatchMessage        = user32.NewProc("DispatchMessageW")
	sendMessage            = user32.NewProc("SendMessageW")
	postMessage            = user32.NewProc("PostMessageW")
	setWindowText          = user32.NewProc("SetWindowTextW")
	getWindowText          = user32.NewProc("GetWindowTextW")
	getWindowTextLength    = user32.NewProc("GetWindowTextLengthW")
	enableWindow           = user32.NewProc("EnableWindow")
	messageBox             = user32.NewProc("MessageBoxW")
	createPopupMenu        = user32.NewProc("CreatePopupMenu")
	appendMenu             = user32.NewProc("AppendMenuW")
	trackPopupMenu         = user32.NewProc("TrackPopupMenu")
	destroyMenu            = user32.NewProc("DestroyMenu")
	getCursorPos           = user32.NewProc("GetCursorPos")
	setTimer               = user32.NewProc("SetTimer")
	killTimer              = user32.NewProc("KillTimer")
	loadCursor             = user32.NewProc("LoadCursorW")
	loadIcon               = user32.NewProc("LoadIconW")
	loadImage              = user32.NewProc("LoadImageW")
	getModuleHandle        = kernel32.NewProc("GetModuleHandleW")
	rtlMoveMemory          = kernel32.NewProc("RtlMoveMemory")
	createMutex            = kernel32.NewProc("CreateMutexW")
	closeHandle            = kernel32.NewProc("CloseHandle")
	getStockObject         = gdi32.NewProc("GetStockObject")
	createFont             = gdi32.NewProc("CreateFontW")
	deleteObject           = gdi32.NewProc("DeleteObject")
	createSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	createRoundRectRgn     = gdi32.NewProc("CreateRoundRectRgn")
	fillRgn                = gdi32.NewProc("FillRgn")
	createCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	selectObject           = gdi32.NewProc("SelectObject")
	deleteDC               = gdi32.NewProc("DeleteDC")
	stretchBlt             = gdi32.NewProc("StretchBlt")
	setStretchBltMode      = gdi32.NewProc("SetStretchBltMode")
	setTextColor           = gdi32.NewProc("SetTextColor")
	setBkColor             = gdi32.NewProc("SetBkColor")
	setBkMode              = gdi32.NewProc("SetBkMode")
	fillRect               = user32.NewProc("FillRect")
	drawText               = user32.NewProc("DrawTextW")
	getOpenFileName        = comdlg32.NewProc("GetOpenFileNameW")
	initCommonControlsEx   = comctl32.NewProc("InitCommonControlsEx")
	setWindowTheme         = uxtheme.NewProc("SetWindowTheme")
	shellNotifyIcon        = shell32.NewProc("Shell_NotifyIconW")
	shellExecute           = shell32.NewProc("ShellExecuteW")
	dwmSetWindowAttribute  = dwmapi.NewProc("DwmSetWindowAttribute")
	regGetValue            = advapi32.NewProc("RegGetValueW")
	winmm                  = syscall.NewLazyDLL("winmm.dll")
	timeBeginPeriod        = winmm.NewProc("timeBeginPeriod")
	timeEndPeriod          = winmm.NewProc("timeEndPeriod")
)

type asyncResult struct {
	kind       string
	message    string
	paused     bool
	candidates []config.Game
	err        error
}

type window struct {
	store   *config.Store
	monitor *app.Monitor
	client  *leigod.Client
	scanner processes.Scanner

	hwnd              uintptr
	font              uintptr
	titleFont         uintptr
	sidebarBg         uintptr
	sidebarBrush      uintptr
	canvasBrush       uintptr
	whiteBrush        uintptr
	panelBrush        uintptr
	shadowBrush       uintptr
	navActiveBrush    uintptr
	inputBorderBrush  uintptr
	sectionFont       uintptr
	sidebarControls   map[uintptr]bool
	panelControls     map[uintptr]bool
	shadowControls    map[uintptr]bool
	navActive         map[uintptr]bool
	navButtons        map[int]uintptr
	toggleControls    map[uintptr]bool
	toggleStates      map[uintptr]bool
	toggleAnimation   map[uintptr]toggleMotion
	toggleTimerActive bool
	inputBorders      map[uintptr]bool
	listHeaderCtrls   map[uintptr]bool
	listHeaders       map[uintptr]bool
	darkMode          bool
	icon              uintptr
	trayAdded         bool
	autoStarted       bool
	currentView       int
	views             map[int][]uintptr
	headerTitle       uintptr

	statusTitle   uintptr
	statusMessage uintptr
	statusMetrics uintptr
	runningList   uintptr
	accountText   uintptr

	gameList         uintptr
	gameName         uintptr
	gameExe          uintptr
	gameSection      uintptr
	categorySummary  uintptr
	candidateList    uintptr
	gameHeaders      []uintptr
	candidateHeaders []uintptr
	candidateAdd     uintptr
	candidateBack    uintptr
	toggleGameButton uintptr
	removeGameButton uintptr
	candidates       []config.Game
	candidateMode    bool

	monitoring       uintptr
	autoPause        uintptr
	autoStart        uintptr
	silentStart      uintptr
	startMinimized   uintptr
	closeAction      uintptr
	checkInterval    uintptr
	gracePeriod      uintptr
	leiGodPath       uintptr
	username         uintptr
	password         uintptr
	token            uintptr
	credentialStatus uintptr

	results chan asyncResult
	busyMu  sync.Mutex
	busy    bool
}

type toggleMotion struct {
	from    float64
	to      float64
	started time.Time
	dur     time.Duration
}

type togglePalette struct {
	offRed, offGreen, offBlue uint32
	onRed, onGreen, onBlue    uint32
	knob                      uintptr
}

var activeWindow *window

func Run(store *config.Store, monitor *app.Monitor, client *leigod.Client, scanner processes.Scanner, autoStarted bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mutexName := utf16Ptr("Local\\LeiGodAutoPauseNativeSingleton")
	mutex, _, mutexErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
	if mutex == 0 {
		return fmt.Errorf("无法创建程序互斥锁: %v", mutexErr)
	}
	defer closeHandle.Call(mutex)
	if errno, ok := mutexErr.(syscall.Errno); ok && errno == errorAlreadyRun {
		className := utf16Ptr(windowClass)
		existing, _, _ := findWindow.Call(uintptr(unsafe.Pointer(className)), 0)
		if existing != 0 {
			showWindow.Call(existing, swRestore)
			setForegroundWindow.Call(existing)
		}
		return nil
	}

	controls := initCommonControls{Size: uint32(unsafe.Sizeof(initCommonControls{})), ICC: 1}
	initCommonControlsEx.Call(uintptr(unsafe.Pointer(&controls)))

	w := &window{store: store, monitor: monitor, client: client, scanner: scanner, views: map[int][]uintptr{}, sidebarControls: map[uintptr]bool{}, panelControls: map[uintptr]bool{}, shadowControls: map[uintptr]bool{}, navActive: map[uintptr]bool{}, navButtons: map[int]uintptr{}, toggleControls: map[uintptr]bool{}, toggleStates: map[uintptr]bool{}, toggleAnimation: map[uintptr]toggleMotion{}, inputBorders: map[uintptr]bool{}, listHeaderCtrls: map[uintptr]bool{}, listHeaders: map[uintptr]bool{}, results: make(chan asyncResult, 8), currentView: viewOverview, autoStarted: autoStarted}
	w.darkMode = systemDarkMode()
	w.rebuildThemeBrushes()
	activeWindow = w
	defer func() { activeWindow = nil }()

	instance, _, _ := getModuleHandle.Call(0)
	cursor, _, _ := loadCursor.Call(0, idcArrow)
	icon := loadAppIcon()
	w.icon = icon
	className := utf16Ptr(windowClass)
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: syscall.NewCallback(windowProc), Instance: instance, Icon: icon, Cursor: cursor, Background: w.canvasBrush, ClassName: className, IconSmall: icon}
	if atom, _, registerErr := registerClassEx.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		return fmt.Errorf("无法注册窗口: %v", registerErr)
	}
	title := utf16Ptr(windowTitle)
	hwnd, _, createErr := createWindowEx.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), wsOverlapped|wsCaption|wsSysMenu|wsMinimizeBox|wsClipChildren, 120, 80, 980, 680, 0, 0, instance, 0)
	if hwnd == 0 {
		return fmt.Errorf("无法创建窗口: %v", createErr)
	}
	w.hwnd = hwnd
	applyWin11Style(hwnd, w.darkMode)
	settings := store.Snapshot()
	if autoStarted && settings.SilentStart {
		w.addTrayIcon()
		showWindow.Call(hwnd, swHide)
	} else if settings.StartMinimized {
		showWindow.Call(hwnd, swShowMinimized)
	} else {
		showWindow.Call(hwnd, swShow)
	}
	updateWindow.Call(hwnd)

	var message msg
	for {
		result, _, callErr := getMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			return fmt.Errorf("窗口消息循环失败: %v", callErr)
		}
		if result == 0 {
			return nil
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func ShowError(title, text string) {
	messageBox.Call(0, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), mbOK|mbIconError)
}

func windowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	w := activeWindow
	if w == nil {
		result, _, _ := defWindowProc.Call(hwnd, uintptr(message), wParam, lParam)
		return result
	}
	switch message {
	case wmCreate:
		w.hwnd = hwnd
		w.build()
		w.applyControlThemes()
		setTimer.Call(hwnd, 1, 1000, 0)
		return 0
	case wmCommand:
		id, notification, source := int(wParam&0xffff), int((wParam>>16)&0xffff), lParam
		if w.toggleControls[source] && notification == bnClicked {
			now := time.Now()
			current := w.toggleProgress(source, now)
			w.toggleStates[source] = !w.toggleStates[source]
			w.startToggleAnimation(source, current, now)
			return 0
		}
		w.command(id, notification, source)
		return 0
	case wmDrawItem:
		if lParam != 0 {
			var item drawItem
			rtlMoveMemory.Call(uintptr(unsafe.Pointer(&item)), lParam, unsafe.Sizeof(item))
			w.drawButton(&item)
			return 1
		}
		return 0
	case wmNotify:
		return w.handleNotify(lParam)
	case wmTimer:
		if wParam == 2 {
			w.advanceToggleAnimations()
		} else {
			w.updateSystemTheme()
			w.updateOverview()
		}
		return 0
	case wmSettingChange:
		w.updateSystemTheme()
		return 0
	case wmSize:
		if wParam == sizeMinimized {
			w.addTrayIcon()
			showWindow.Call(hwnd, swHide)
		}
		return 0
	case wmTray:
		event := uint32(lParam & 0xffff)
		if event == wmLButtonUp || event == wmLButtonDbl {
			w.removeTrayIcon()
			showWindow.Call(hwnd, swRestore)
			setForegroundWindow.Call(hwnd)
		} else if event == wmRButtonUp || event == wmContextMenu {
			w.showTrayMenu()
		}
		return 0
	case wmAppResult:
		w.handleResults()
		return 0
	case wmCtlColorStatic, wmCtlColorBtn:
		if w.listHeaderCtrls[lParam] {
			setTextColor.Call(wParam, w.textColor())
			setBkMode.Call(wParam, 1)
			return w.whiteBrush
		}
		if w.inputBorders[lParam] {
			setBkMode.Call(wParam, 1)
			return w.inputBorderBrush
		}
		if w.sidebarControls[lParam] {
			if w.navActive[lParam] {
				setTextColor.Call(wParam, w.accentTextColor())
			} else {
				setTextColor.Call(wParam, w.textColor())
			}
			setBkMode.Call(wParam, 1)
			if w.navActive[lParam] {
				return w.navActiveBrush
			}
			return w.sidebarBrush
		}
		if w.shadowControls[lParam] {
			setBkMode.Call(wParam, 1)
			return w.shadowBrush
		}
		if w.panelControls[lParam] {
			setTextColor.Call(wParam, w.textColor())
			setBkMode.Call(wParam, 1)
			return w.panelBrush
		}
		setTextColor.Call(wParam, w.textColor())
		setBkMode.Call(wParam, 1)
		return w.canvasBrush
	case wmCtlColorEdit, wmCtlColorList:
		setTextColor.Call(wParam, w.textColor())
		setBkMode.Call(wParam, 1)
		return w.whiteBrush
	case wmClose:
		w.handleClose()
		return 0
	case wmDestroy:
		killTimer.Call(hwnd, 1)
		w.stopToggleTimer()
		w.removeTrayIcon()
		if w.font != 0 {
			deleteObject.Call(w.font)
		}
		if w.titleFont != 0 {
			deleteObject.Call(w.titleFont)
		}
		if w.sectionFont != 0 {
			deleteObject.Call(w.sectionFont)
		}
		if w.sidebarBrush != 0 {
			deleteObject.Call(w.sidebarBrush)
		}
		if w.canvasBrush != 0 {
			deleteObject.Call(w.canvasBrush)
		}
		if w.whiteBrush != 0 {
			deleteObject.Call(w.whiteBrush)
		}
		if w.panelBrush != 0 {
			deleteObject.Call(w.panelBrush)
		}
		if w.shadowBrush != 0 {
			deleteObject.Call(w.shadowBrush)
		}
		if w.navActiveBrush != 0 {
			deleteObject.Call(w.navActiveBrush)
		}
		if w.inputBorderBrush != 0 {
			deleteObject.Call(w.inputBorderBrush)
		}
		postQuitMessage.Call(0)
		return 0
	}
	result, _, _ := defWindowProc.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func (w *window) build() {
	w.font = makeFont(-15, 400)
	w.titleFont = makeFont(-24, 600)
	w.sectionFont = makeFont(-16, 600)
	w.sidebarBg = w.create("STATIC", "", wsChild|wsVisible, 0, 0, 190, 680, 0, 0)
	brand := w.create("STATIC", "雷神自动暂停", wsChild|wsVisible, 24, 24, 150, 30, 0, 0)
	setFont(brand, w.titleFont)
	w.create("STATIC", "LEIGOD CONTROL", wsChild|wsVisible, 24, 52, 150, 18, 0, 0)
	w.navButtons[viewOverview] = w.create("BUTTON", "概览", wsChild|wsVisible|wsTabStop|bsPushButton|bsFlat, 20, 96, 150, 42, idNavOverview, 0)
	w.navButtons[viewGames] = w.create("BUTTON", "游戏监控", wsChild|wsVisible|wsTabStop|bsPushButton|bsFlat, 20, 146, 150, 42, idNavGames, 0)
	w.navButtons[viewSettings] = w.create("BUTTON", "设置", wsChild|wsVisible|wsTabStop|bsPushButton|bsFlat, 20, 196, 150, 42, idNavSettings, 0)
	w.create("STATIC", "低频检测  ·  本机运行", wsChild|wsVisible, 20, 610, 165, 22, 0, 0)
	w.headerTitle = w.create("STATIC", "概览", wsChild|wsVisible, 210, 26, 710, 30, 0, 0)
	setFont(w.headerTitle, w.titleFont)

	w.buildOverview()
	w.buildGames()
	w.buildSettings()
	w.switchView(viewOverview)
	w.updateOverview()
	w.refreshGames()
	w.fillSettings()
}

func (w *window) buildOverview() {
	w.createPanel(viewOverview, "当前状态", 210, 80, 730, 142)
	w.statusTitle = w.create("STATIC", "等待游戏运行", wsChild, 235, 112, 660, 28, 0, viewOverview)
	setFont(w.statusTitle, w.titleFont)
	w.add(viewOverview, w.statusTitle)
	w.statusMessage = w.create("STATIC", "监控服务已启动", wsChild, 235, 146, 660, 24, 0, viewOverview)
	w.add(viewOverview, w.statusMessage)
	w.statusMetrics = w.create("STATIC", "", wsChild, 235, 178, 660, 22, 0, viewOverview)
	w.add(viewOverview, w.statusMetrics)

	w.createPanel(viewOverview, "运行中的游戏", 210, 238, 465, 330)
	w.runningList = w.create("LISTBOX", "", wsChild|wsBorder|wsVScroll|lbsNotify|lbsNoIntegral, 230, 272, 425, 274, 0, viewOverview)
	w.add(viewOverview, w.runningList)
	w.createPanel(viewOverview, "雷神时长", 695, 238, 245, 330)
	w.accountText = w.create("STATIC", "尚未查询雷神账号状态", wsChild, 717, 278, 200, 48, 0, viewOverview)
	w.add(viewOverview, w.accountText)
	w.add(viewOverview, w.create("BUTTON", "查询账号", wsChild|wsTabStop|bsPushButton, 717, 344, 200, 38, idCheckAccount, viewOverview))
	w.add(viewOverview, w.create("BUTTON", "立即暂停时长", wsChild|wsTabStop|bsPushButton, 717, 394, 200, 42, idPauseNow, viewOverview))
	w.add(viewOverview, w.create("BUTTON", "打开雷神加速器", wsChild|wsTabStop|bsPushButton, 717, 448, 200, 38, idOpenLeiGod, viewOverview))
}

func (w *window) buildGames() {
	w.gameSection = w.createPanel(viewGames, "已监控游戏", 210, 68, 730, 400)
	w.categorySummary = w.create("STATIC", "", wsChild, 234, 108, 390, 20, 0, viewGames)
	w.add(viewGames, w.categorySummary)
	w.add(viewGames, w.create("BUTTON", "扫描运行进程", wsChild|wsTabStop|bsPushButton, 650, 78, 135, 36, idScanRunning, viewGames))
	w.add(viewGames, w.create("BUTTON", "扫描 Steam / Epic", wsChild|wsTabStop|bsPushButton, 795, 78, 145, 36, idScanInstalled, viewGames))
	gameColumns := []columnDef{{"状态", 70}, {"游戏", 210}, {"进程", 210}, {"来源", 180}}
	w.gameHeaders = w.createListHeader(viewGames, 210, 136, 730, gameColumns)
	w.gameList = w.create("SysListView32", "", wsChild|wsBorder|wsTabStop|lvsReport|lvsSingleSel|lvsShowSelect|lvsNoHeader|lvsNoSortHead, 210, 160, 730, 292, idGameList, viewGames)
	w.add(viewGames, w.gameList)
	listViewColumns(w.gameList, gameColumns)
	candidateColumns := []columnDef{{"分类", 100}, {"进程", 155}, {"路径", 365}, {"来源", 90}}
	w.candidateHeaders = w.createListHeader(viewGames, 210, 136, 730, candidateColumns)
	w.candidateList = w.create("SysListView32", "", wsChild|wsBorder|wsTabStop|lvsReport|lvsSingleSel|lvsShowSelect|lvsNoHeader|lvsNoSortHead, 210, 160, 730, 292, idCandidateList, viewGames)
	w.add(viewGames, w.candidateList)
	listViewColumns(w.candidateList, candidateColumns)
	w.candidateAdd = w.create("BUTTON", "添加选中项", wsChild|wsTabStop|bsPushButton, 210, 462, 130, 36, idCandidateAdd, viewGames)
	w.candidateBack = w.create("BUTTON", "返回监控列表", wsChild|wsTabStop|bsPushButton, 350, 462, 130, 36, idCandidateBack, viewGames)
	w.add(viewGames, w.candidateAdd, w.candidateBack)
	w.toggleGameButton = w.create("BUTTON", "启用 / 停用", wsChild|wsTabStop|bsPushButton, 210, 462, 120, 36, idToggleGame, viewGames)
	w.removeGameButton = w.create("BUTTON", "移除", wsChild|wsTabStop|bsPushButton, 340, 462, 90, 36, idRemoveGame, viewGames)
	w.add(viewGames, w.toggleGameButton, w.removeGameButton)

	w.createPanel(viewGames, "手动添加游戏", 210, 500, 730, 132)
	w.add(viewGames, w.create("STATIC", "游戏名称", wsChild, 210, 546, 90, 20, 0, viewGames))
	w.gameName = w.createInput("", wsChild|wsTabStop|esAutoHScroll, 210, 568, 220, 34, viewGames)
	w.add(viewGames, w.gameName)
	w.add(viewGames, w.create("STATIC", "进程名（例如 game.exe）", wsChild, 450, 546, 190, 20, 0, viewGames))
	w.gameExe = w.createInput("", wsChild|wsTabStop|esAutoHScroll, 450, 568, 250, 34, viewGames)
	w.add(viewGames, w.gameExe)
	w.add(viewGames, w.create("BUTTON", "选择 EXE", wsChild|wsTabStop|bsPushButton, 712, 566, 100, 36, idBrowseGame, viewGames))
	w.add(viewGames, w.create("BUTTON", "添加", wsChild|wsTabStop|bsPushButton, 824, 566, 116, 36, idAddManual, viewGames))
	w.showCandidateMode(false)
}

func (w *window) buildSettings() {
	w.createPanel(viewSettings, "自动化", 210, 78, 730, 180)
	w.monitoring = w.create("BUTTON", "启用进程监控", wsChild|wsTabStop|bsAutoCheckbox, 235, 112, 210, 28, 0, viewSettings)
	w.autoPause = w.create("BUTTON", "游戏退出后自动暂停", wsChild|wsTabStop|bsAutoCheckbox, 235, 148, 220, 28, 0, viewSettings)
	w.autoStart = w.create("BUTTON", "开机自动启动", wsChild|wsTabStop|bsAutoCheckbox, 510, 112, 210, 28, 0, viewSettings)
	w.silentStart = w.create("BUTTON", "开机静默启动到托盘", wsChild|wsTabStop|bsAutoCheckbox, 510, 148, 230, 28, 0, viewSettings)
	w.startMinimized = w.create("BUTTON", "启动时最小化窗口", wsChild|wsTabStop|bsAutoCheckbox, 510, 184, 230, 28, 0, viewSettings)
	w.add(viewSettings, w.monitoring, w.autoPause, w.autoStart, w.silentStart, w.startMinimized)
	w.add(viewSettings, w.create("STATIC", "关闭窗口", wsChild, 235, 185, 90, 22, 0, viewSettings))
	w.closeAction = w.create("COMBOBOX", "", wsChild|wsTabStop|wsVScroll|cbsDropDownList|cbsOwnerDraw|cbsHasStrings, 325, 180, 170, 140, 0, viewSettings)
	w.add(viewSettings, w.closeAction)
	comboAdd(w.closeAction, "每次询问")
	comboAdd(w.closeAction, "隐藏到系统托盘")
	comboAdd(w.closeAction, "直接退出程序")
	w.add(viewSettings, w.create("STATIC", "检测间隔（2-60 秒）", wsChild, 235, 222, 190, 22, 0, viewSettings))
	w.checkInterval = w.createInput("3", wsChild|wsTabStop|esNumber, 405, 217, 110, 30, viewSettings)
	w.add(viewSettings, w.checkInterval)
	w.add(viewSettings, w.create("STATIC", "退出宽限（0-3600 秒）", wsChild, 550, 222, 195, 22, 0, viewSettings))
	w.gracePeriod = w.createInput("30", wsChild|wsTabStop|esNumber, 760, 217, 110, 30, viewSettings)
	w.add(viewSettings, w.gracePeriod)

	w.createPanel(viewSettings, "雷神加速器", 210, 280, 730, 300)
	w.add(viewSettings, w.create("STATIC", "雷神程序路径", wsChild, 235, 316, 130, 22, 0, viewSettings))
	w.leiGodPath = w.createInput("", wsChild|wsTabStop|esAutoHScroll, 375, 312, 315, 32, viewSettings)
	w.add(viewSettings, w.leiGodPath)
	w.add(viewSettings, w.create("BUTTON", "选择程序", wsChild|wsTabStop|bsPushButton, 702, 310, 96, 36, idBrowseLeiGod, viewSettings))
	w.add(viewSettings, w.create("BUTTON", "自动查找", wsChild|wsTabStop|bsPushButton, 810, 310, 100, 36, idFindLeiGod, viewSettings))
	w.add(viewSettings, w.create("STATIC", "手机号 / 用户名", wsChild, 235, 362, 130, 22, 0, viewSettings))
	w.username = w.createInput("", wsChild|wsTabStop|esAutoHScroll, 375, 358, 260, 32, viewSettings)
	w.add(viewSettings, w.username)
	w.add(viewSettings, w.create("STATIC", "密码（留空则不修改）", wsChild, 235, 408, 165, 22, 0, viewSettings))
	w.password = w.createInput("", wsChild|wsTabStop|esAutoHScroll|esPassword, 405, 404, 230, 32, viewSettings)
	w.add(viewSettings, w.password)
	w.add(viewSettings, w.create("BUTTON", "清除密码", wsChild|wsTabStop|bsPushButton, 650, 402, 105, 35, idClearPassword, viewSettings))
	w.add(viewSettings, w.create("STATIC", "账号 Token（留空不修改）", wsChild, 235, 454, 215, 22, 0, viewSettings))
	w.token = w.createInput("", wsChild|wsTabStop|esAutoHScroll|esPassword, 455, 450, 290, 32, viewSettings)
	w.add(viewSettings, w.token)
	w.add(viewSettings, w.create("BUTTON", "清除 Token", wsChild|wsTabStop|bsPushButton, 760, 448, 120, 35, idClearToken, viewSettings))
	w.credentialStatus = w.create("STATIC", "", wsChild, 235, 504, 440, 24, 0, viewSettings)
	w.add(viewSettings, w.credentialStatus)
	w.add(viewSettings, w.create("BUTTON", "查询账号", wsChild|wsTabStop|bsPushButton, 690, 502, 100, 36, idSettingsAccount, viewSettings))
	w.add(viewSettings, w.create("BUTTON", "保存设置", wsChild|wsTabStop|bsPushButton, 805, 502, 105, 36, idSaveSettings, viewSettings))
}

func (w *window) create(class, text string, style uint32, x, y, width, height, id, view int) uintptr {
	lowerClass := strings.ToLower(class)
	isToggle := lowerClass == "button" && style&0x0f == bsAutoCheckbox
	if lowerClass == "button" && style&0x0f == bsPushButton {
		style |= bsOwnerDraw
	}
	if isToggle {
		style |= bsOwnerDraw
	}
	if lowerClass == "edit" || lowerClass == "listbox" || lowerClass == "syslistview32" {
		style &^= wsBorder
	}
	classPtr, textPtr := utf16Ptr(class), utf16Ptr(text)
	hwnd, _, _ := createWindowEx.Call(0, uintptr(unsafe.Pointer(classPtr)), uintptr(unsafe.Pointer(textPtr)), uintptr(style), uintptr(x), uintptr(y), uintptr(width), uintptr(height), w.hwnd, uintptr(id), 0, 0)
	if hwnd != 0 && w.font != 0 {
		sendMessage.Call(hwnd, wmSetFont, w.font, 1)
		themeName := "Explorer"
		if w.darkMode {
			themeName = "DarkMode_Explorer"
		}
		setWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(themeName))), 0)
	}
	if hwnd != 0 && x < 190 {
		w.sidebarControls[hwnd] = true
		empty := utf16Ptr("")
		setWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(empty)), uintptr(unsafe.Pointer(empty)))
	}
	if hwnd != 0 && view != 0 && x >= 190 && y >= 68 {
		w.panelControls[hwnd] = true
	}
	if hwnd != 0 && isToggle {
		w.toggleControls[hwnd] = true
	}
	if hwnd != 0 {
		if (lowerClass == "button" && style&0x0f == bsOwnerDraw) || lowerClass == "edit" || lowerClass == "combobox" || lowerClass == "listbox" || lowerClass == "syslistview32" {
			radius := 14
			if lowerClass == "syslistview32" {
				radius = 16
			}
			roundControl(hwnd, width, height, radius)
		}
	}
	return hwnd
}

func (w *window) createInput(text string, style uint32, x, y, width, height, view int) uintptr {
	border := w.create("STATIC", "", wsChild, x-2, y-2, width+4, height+4, 0, view)
	if border != 0 {
		w.inputBorders[border] = true
		roundControl(border, width+4, height+4, 16)
		w.add(view, border)
	}
	return w.create("EDIT", text, style, x, y, width, height, 0, view)
}

func (w *window) createListHeader(view, x, y, width int, columns []columnDef) []uintptr {
	background := w.create("STATIC", "", wsChild, x, y, width, 24, 0, view)
	controls := []uintptr{background}
	if background != 0 {
		w.listHeaderCtrls[background] = true
		roundControl(background, width, 24, 12)
		w.add(view, background)
	}
	offset := 0
	for _, column := range columns {
		label := w.create("STATIC", column.Title, wsChild, x+offset+8, y+3, int(column.Width)-12, 20, 0, view)
		if label != 0 {
			w.listHeaderCtrls[label] = true
			w.add(view, label)
			controls = append(controls, label)
		}
		offset += int(column.Width)
	}
	return controls
}

func (w *window) createPanel(view int, title string, x, y, width, height int) uintptr {
	shadow := w.create("STATIC", "", wsChild|wsVisible, x+6, y+7, width, height, 0, view)
	if shadow != 0 {
		w.shadowControls[shadow] = true
		roundControl(shadow, width, height, 22)
	}
	panel := w.create("STATIC", "", wsChild|wsVisible, x, y, width, height, 0, view)
	if panel != 0 {
		w.panelControls[panel] = true
		roundControl(panel, width, height, 22)
	}
	label := w.create("STATIC", title, wsChild|wsVisible, x+24, y+14, width-48, 26, 0, view)
	if label != 0 {
		w.panelControls[label] = true
		setFont(label, w.sectionFont)
	}
	w.add(view, shadow, panel, label)
	return label
}

func (w *window) drawToggle(item *drawItem) {
	background := w.panelBrush
	if w.sidebarControls[item.HWndItem] {
		background = w.sidebarBrush
	}
	fillRect.Call(item.HDC, uintptr(unsafe.Pointer(&item.Rect)), background)
	left, top := item.Rect.Left+4, item.Rect.Top+3
	eased := w.toggleProgress(item.HWndItem, time.Now())
	palette := w.togglePalette()
	trackColor := rgb(
		mixChannelUnit(palette.offRed, palette.onRed, eased),
		mixChannelUnit(palette.offGreen, palette.onGreen, eased),
		mixChannelUnit(palette.offBlue, palette.onBlue, eased),
	)
	w.drawToggleTrack(item.HDC, background, left, top, eased, trackColor, palette.knob)
	length, _, _ := getWindowTextLength.Call(item.HWndItem)
	if length == 0 {
		return
	}
	textBuffer := make([]uint16, length+1)
	getWindowText.Call(item.HWndItem, uintptr(unsafe.Pointer(&textBuffer[0])), length+1)
	setTextColor.Call(item.HDC, w.textColor())
	setBkMode.Call(item.HDC, 1)
	textRect := item.Rect
	textRect.Left += 58
	drawText.Call(item.HDC, uintptr(unsafe.Pointer(&textBuffer[0])), length, uintptr(unsafe.Pointer(&textRect)), 4|0x20|0x800)
}

func (w *window) togglePalette() togglePalette {
	if w.darkMode {
		return togglePalette{
			offRed: 86, offGreen: 92, offBlue: 101,
			onRed: 56, onGreen: 157, onBlue: 205,
			knob: rgb(236, 240, 244),
		}
	}
	return togglePalette{
		offRed: 211, offGreen: 220, offBlue: 229,
		onRed: 76, onGreen: 177, onBlue: 219,
		knob: rgb(250, 252, 254),
	}
}

func (w *window) drawToggleTrack(hdc, background uintptr, left, top int32, progress float64, trackColor, knobColor uintptr) {
	sourceWidth := toggleTrackWidth * toggleSupersample
	sourceHeight := toggleTrackHeight * toggleSupersample
	memoryDC, _, _ := createCompatibleDC.Call(hdc)
	if memoryDC == 0 {
		return
	}
	defer deleteDC.Call(memoryDC)

	bitmap, _, _ := createCompatibleBitmap.Call(hdc, uintptr(sourceWidth), uintptr(sourceHeight))
	if bitmap == 0 {
		return
	}
	oldBitmap, _, _ := selectObject.Call(memoryDC, bitmap)
	defer func() {
		selectObject.Call(memoryDC, oldBitmap)
		deleteObject.Call(bitmap)
	}()

	sourceRect := rect{Right: sourceWidth, Bottom: sourceHeight}
	fillRect.Call(memoryDC, uintptr(unsafe.Pointer(&sourceRect)), background)

	trackBrush := solidBrush(trackColor)
	track, _, _ := createRoundRectRgn.Call(0, 0, uintptr(sourceWidth), uintptr(sourceHeight), uintptr(sourceHeight), uintptr(sourceHeight))
	if track != 0 {
		fillRgn.Call(memoryDC, track, trackBrush)
		deleteObject.Call(track)
	}
	if trackBrush != 0 {
		deleteObject.Call(trackBrush)
	}

	knobLeft := int32((3+float64(toggleKnobTravel)*progress)*float64(toggleSupersample) + 0.5)
	knobTop := int32(3) * toggleSupersample
	knobSize := toggleKnobSize * toggleSupersample
	knob, _, _ := createRoundRectRgn.Call(uintptr(knobLeft), uintptr(knobTop), uintptr(knobLeft+knobSize), uintptr(knobTop+knobSize), uintptr(knobSize), uintptr(knobSize))
	knobBrush := solidBrush(knobColor)
	if knob != 0 {
		fillRgn.Call(memoryDC, knob, knobBrush)
		deleteObject.Call(knob)
	}
	if knobBrush != 0 {
		deleteObject.Call(knobBrush)
	}

	setStretchBltMode.Call(hdc, stretchModeHalftone)
	stretchBlt.Call(hdc, uintptr(left), uintptr(top), uintptr(toggleTrackWidth), uintptr(toggleTrackHeight), memoryDC, 0, 0, uintptr(sourceWidth), uintptr(sourceHeight), rasterCopy)
}

func (w *window) handleNotify(pointer uintptr) uintptr {
	if pointer == 0 {
		return 0
	}
	var header notifyHeader
	rtlMoveMemory.Call(uintptr(unsafe.Pointer(&header)), pointer, unsafe.Sizeof(header))
	if header.Code != nmCustomDraw {
		return 0
	}
	var base baseCustomDraw
	rtlMoveMemory.Call(uintptr(unsafe.Pointer(&base)), pointer, unsafe.Sizeof(base))
	if base.DrawStage == cdsPrepaint {
		return cdrfNotifyItem
	}
	if base.DrawStage == cdsItemPrepaint && w.darkMode && w.listHeaders[base.Header.HWndFrom] {
		w.drawListHeader(&base)
		return cdrfSkipDefault
	}
	if base.DrawStage == cdsItemPrepaint {
		var draw customDraw
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&draw)), pointer, unsafe.Sizeof(draw))
		background := w.listRowColor(draw.Header.HWndFrom, int(draw.ItemSpec))
		if background == 0 {
			return 0
		}
		// ListView uses clrTextBk for the actual item fill; SetBkColor alone only
		// affects text rendering on some Windows themes.
		textBkOffset := unsafe.Offsetof(customDraw{}.TextBk)
		rtlMoveMemory.Call(pointer+textBkOffset, uintptr(unsafe.Pointer(&background)), unsafe.Sizeof(background))
		setBkColor.Call(draw.HDC, background)
		setTextColor.Call(draw.HDC, w.textColor())
		return cdrfNewFont
	}
	return 0
}

func (w *window) drawListHeader(draw *baseCustomDraw) {
	if draw == nil || draw.HDC == 0 {
		return
	}
	fillRect.Call(draw.HDC, uintptr(unsafe.Pointer(&draw.Rect)), w.panelBrush)
	buffer := make([]uint16, 128)
	item := headerItem{Mask: hdiText, Text: &buffer[0], TextLength: int32(len(buffer))}
	sendMessage.Call(draw.Header.HWndFrom, hdmGetItem, draw.ItemSpec, uintptr(unsafe.Pointer(&item)))
	setTextColor.Call(draw.HDC, w.textColor())
	setBkMode.Call(draw.HDC, 1)
	textRect := draw.Rect
	textRect.Left += 8
	textRect.Right -= 6
	drawText.Call(draw.HDC, uintptr(unsafe.Pointer(&buffer[0])), ^uintptr(0), uintptr(unsafe.Pointer(&textRect)), 4|0x20|0x800)
	separator := draw.Rect
	separator.Left = separator.Right - 1
	fillRect.Call(draw.HDC, uintptr(unsafe.Pointer(&separator)), w.inputBorderBrush)
}

func (w *window) listRowColor(hwnd uintptr, row int) uintptr {
	if hwnd != w.gameList && hwnd != w.candidateList {
		return 0
	}
	if w.darkMode {
		if row%2 == 0 {
			return rgb(49, 52, 58)
		}
		return rgb(62, 69, 77)
	}
	if row%2 == 0 {
		return rgb(236, 244, 250)
	}
	return rgb(203, 220, 235)
}

func (w *window) drawButton(item *drawItem) {
	if item == nil || item.HDC == 0 || item.HWndItem == 0 {
		return
	}
	if item.HWndItem == w.closeAction {
		w.drawComboItem(item)
		return
	}
	if w.toggleControls[item.HWndItem] {
		w.drawToggle(item)
		return
	}
	background := w.panelBrush
	if w.sidebarControls[item.HWndItem] {
		background = w.sidebarBrush
	}
	fillRect.Call(item.HDC, uintptr(unsafe.Pointer(&item.Rect)), background)

	left, top := item.Rect.Left, item.Rect.Top
	right, bottom := item.Rect.Right, item.Rect.Bottom
	shadowRegion, _, _ := createRoundRectRgn.Call(uintptr(left+2), uintptr(top+3), uintptr(right-1), uintptr(bottom), 14, 14)
	if shadowRegion != 0 {
		fillRgn.Call(item.HDC, shadowRegion, w.shadowBrush)
		deleteObject.Call(shadowRegion)
	}

	mainColor := rgb(247, 250, 252)
	if w.darkMode {
		mainColor = rgb(55, 58, 64)
	}
	if w.navActive[item.HWndItem] {
		if w.darkMode {
			mainColor = rgb(53, 72, 83)
		} else {
			mainColor = rgb(216, 234, 246)
		}
	} else if item.ItemState&odsSelected != 0 {
		if w.darkMode {
			mainColor = rgb(67, 71, 78)
		} else {
			mainColor = rgb(222, 231, 239)
		}
	}
	if item.ItemState&odsDisabled != 0 {
		if w.darkMode {
			mainColor = rgb(48, 50, 55)
		} else {
			mainColor = rgb(224, 229, 235)
		}
	}
	mainBrush := solidBrush(mainColor)
	mainRegion, _, _ := createRoundRectRgn.Call(uintptr(left), uintptr(top), uintptr(right-2), uintptr(bottom-3), 14, 14)
	if mainRegion != 0 {
		fillRgn.Call(item.HDC, mainRegion, mainBrush)
		deleteObject.Call(mainRegion)
	}
	if mainBrush != 0 {
		deleteObject.Call(mainBrush)
	}

	length, _, _ := getWindowTextLength.Call(item.HWndItem)
	if length == 0 {
		return
	}
	textBuffer := make([]uint16, length+1)
	getWindowText.Call(item.HWndItem, uintptr(unsafe.Pointer(&textBuffer[0])), length+1)
	textColor := w.textColor()
	if w.navActive[item.HWndItem] {
		textColor = w.accentTextColor()
	}
	if item.ItemState&odsDisabled != 0 {
		if w.darkMode {
			textColor = rgb(121, 126, 134)
		} else {
			textColor = rgb(135, 145, 156)
		}
	}
	setTextColor.Call(item.HDC, textColor)
	setBkMode.Call(item.HDC, 1)
	textRect := item.Rect
	textRect.Right -= 2
	textRect.Bottom -= 3
	drawText.Call(item.HDC, uintptr(unsafe.Pointer(&textBuffer[0])), length, uintptr(unsafe.Pointer(&textRect)), 1|4|0x20|0x800)
}

func (w *window) drawComboItem(item *drawItem) {
	background := w.whiteBrush
	textColor := w.textColor()
	if item.ItemState&odsSelected != 0 {
		background = w.navActiveBrush
		textColor = w.accentTextColor()
	}
	fillRect.Call(item.HDC, uintptr(unsafe.Pointer(&item.Rect)), background)
	if item.ItemID == ^uint32(0) {
		return
	}
	length, _, _ := sendMessage.Call(item.HWndItem, cbGetTextLength, uintptr(item.ItemID), 0)
	if int32(length) < 0 {
		return
	}
	buffer := make([]uint16, length+1)
	sendMessage.Call(item.HWndItem, cbGetText, uintptr(item.ItemID), uintptr(unsafe.Pointer(&buffer[0])))
	setTextColor.Call(item.HDC, textColor)
	setBkMode.Call(item.HDC, 1)
	textRect := item.Rect
	textRect.Left += 8
	textRect.Right -= 6
	drawText.Call(item.HDC, uintptr(unsafe.Pointer(&buffer[0])), length, uintptr(unsafe.Pointer(&textRect)), 4|0x20|0x800)
}

func makeFont(height, weight int32) uintptr {
	face := utf16Ptr("Segoe UI")
	font, _, _ := createFont.Call(uintptr(uint32(height)), 0, 0, 0, uintptr(uint32(weight)), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	return font
}

func rgb(red, green, blue uint32) uintptr { return uintptr(red | green<<8 | blue<<16) }
func solidBrush(color uintptr) uintptr    { brush, _, _ := createSolidBrush.Call(color); return brush }

func mixChannel(from, to uint32, progress int) uint32 {
	return uint32((int(from)*(100-progress) + int(to)*progress) / 100)
}

func mixChannelUnit(from, to uint32, progress float64) uint32 {
	return uint32(float64(from)*(1-progress) + float64(to)*progress + 0.5)
}

func systemDarkMode() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LEIGOD_THEME"))) {
	case "dark":
		return true
	case "light":
		return false
	}
	subkey := utf16Ptr(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)
	valueName := utf16Ptr("AppsUseLightTheme")
	value, size := uint32(1), uint32(unsafe.Sizeof(uint32(0)))
	hkeyCurrentUser := ^uintptr(0x7ffffffe)
	result, _, _ := regGetValue.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(subkey)), uintptr(unsafe.Pointer(valueName)), 0x18, 0, uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&size)))
	return result == 0 && value == 0
}

func (w *window) textColor() uintptr {
	if w.darkMode {
		return rgb(226, 229, 233)
	}
	return rgb(44, 56, 70)
}

func (w *window) accentTextColor() uintptr {
	if w.darkMode {
		return rgb(101, 194, 231)
	}
	return rgb(22, 119, 177)
}

func (w *window) rebuildThemeBrushes() {
	old := []uintptr{w.sidebarBrush, w.canvasBrush, w.whiteBrush, w.panelBrush, w.shadowBrush, w.navActiveBrush, w.inputBorderBrush}
	if w.darkMode {
		w.sidebarBrush = solidBrush(rgb(35, 37, 41))
		w.canvasBrush = solidBrush(rgb(28, 30, 34))
		w.whiteBrush = solidBrush(rgb(49, 52, 58))
		w.panelBrush = solidBrush(rgb(41, 43, 48))
		w.shadowBrush = solidBrush(rgb(17, 18, 20))
		w.navActiveBrush = solidBrush(rgb(51, 69, 80))
		w.inputBorderBrush = solidBrush(rgb(92, 98, 108))
	} else {
		w.sidebarBrush = solidBrush(rgb(226, 232, 239))
		w.canvasBrush = solidBrush(rgb(232, 237, 243))
		w.whiteBrush = solidBrush(rgb(249, 251, 253))
		w.panelBrush = solidBrush(rgb(241, 245, 249))
		w.shadowBrush = solidBrush(rgb(205, 213, 222))
		w.navActiveBrush = solidBrush(rgb(216, 234, 246))
		w.inputBorderBrush = solidBrush(rgb(167, 181, 195))
	}
	if w.hwnd != 0 {
		setClassLongPtr.Call(w.hwnd, gclpBackground, w.canvasBrush)
		applyWin11Style(w.hwnd, w.darkMode)
		w.applyControlThemes()
		redrawWindow.Call(w.hwnd, 0, 0, 0x0001|0x0004|0x0080|0x0100)
	}
	for _, brush := range old {
		if brush != 0 {
			deleteObject.Call(brush)
		}
	}
}

func (w *window) updateSystemTheme() {
	dark := systemDarkMode()
	if dark == w.darkMode {
		return
	}
	w.darkMode = dark
	w.rebuildThemeBrushes()
}

func (w *window) applyControlThemes() {
	if w.hwnd == 0 {
		return
	}
	themeName := "Explorer"
	if w.darkMode {
		themeName = "DarkMode_Explorer"
	}
	theme := utf16Ptr(themeName)
	for _, control := range []uintptr{w.gameList, w.candidateList, w.runningList, w.gameName, w.gameExe, w.checkInterval, w.gracePeriod, w.leiGodPath, w.username, w.password, w.token} {
		if control != 0 {
			setWindowTheme.Call(control, uintptr(unsafe.Pointer(theme)), 0)
		}
	}
	if w.closeAction != 0 {
		comboTheme := themeName
		if w.darkMode {
			comboTheme = "DarkMode_CFD"
		}
		comboThemePtr := utf16Ptr(comboTheme)
		setWindowTheme.Call(w.closeAction, uintptr(unsafe.Pointer(comboThemePtr)), 0)
	}
	for _, list := range []uintptr{w.gameList, w.candidateList} {
		if list == 0 {
			continue
		}
		background := w.listRowColor(list, 0)
		sendMessage.Call(list, lvmSetBkColor, 0, background)
		sendMessage.Call(list, lvmSetTextBk, 0, background)
		sendMessage.Call(list, lvmSetTextColor, 0, w.textColor())
		header, _, _ := sendMessage.Call(list, lvmGetHeader, 0, 0)
		if header != 0 {
			w.listHeaders[header] = true
			headerTheme := theme
			if w.darkMode {
				headerTheme = utf16Ptr("DarkMode_ItemsView")
			}
			setWindowTheme.Call(header, uintptr(unsafe.Pointer(headerTheme)), 0)
		}
	}
}

func applyWin11Style(hwnd uintptr, dark bool) {
	cornerPreference := uint32(2) // DWMWCP_ROUND
	dwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&cornerPreference)), unsafe.Sizeof(cornerPreference))
	darkValue := uint32(0)
	captionColor := uint32(rgb(232, 237, 243))
	textColor := uint32(rgb(44, 56, 70))
	if dark {
		darkValue = 1
		captionColor = uint32(rgb(28, 30, 34))
		textColor = uint32(rgb(226, 229, 233))
	}
	dwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&darkValue)), unsafe.Sizeof(darkValue))
	dwmSetWindowAttribute.Call(hwnd, 35, uintptr(unsafe.Pointer(&captionColor)), unsafe.Sizeof(captionColor))
	dwmSetWindowAttribute.Call(hwnd, 36, uintptr(unsafe.Pointer(&textColor)), unsafe.Sizeof(textColor))
}

func roundControl(hwnd uintptr, width, height, radius int) {
	region, _, _ := createRoundRectRgn.Call(0, 0, uintptr(width+1), uintptr(height+1), uintptr(radius), uintptr(radius))
	if region == 0 {
		return
	}
	result, _, _ := setWindowRgn.Call(hwnd, region, 1)
	if result == 0 {
		deleteObject.Call(region)
	}
}

func setFont(hwnd, font uintptr) {
	if hwnd != 0 && font != 0 {
		sendMessage.Call(hwnd, wmSetFont, font, 1)
	}
}

func (w *window) add(view int, controls ...uintptr) {
	w.views[view] = append(w.views[view], controls...)
}

func showControls(controls []uintptr, command uintptr) {
	for _, control := range controls {
		if control != 0 {
			showWindow.Call(control, command)
		}
	}
}

func (w *window) switchView(view int) {
	w.currentView = view
	for currentView, button := range w.navButtons {
		w.navActive[button] = currentView == view
		redrawWindow.Call(button, 0, 0, 0x0001|0x0004)
	}
	for _, controls := range w.views {
		for _, control := range controls {
			if control != 0 {
				showWindow.Call(control, swHide)
			}
		}
	}
	for _, control := range w.views[view] {
		if control != 0 {
			showWindow.Call(control, swShow)
		}
	}
	titles := map[int]string{viewOverview: "概览", viewGames: "游戏监控", viewSettings: "设置"}
	setText(w.headerTitle, titles[view])
	if view == viewGames {
		w.showCandidateMode(w.candidateMode)
	}
	if view == viewSettings {
		w.fillSettings()
	}
	redrawWindow.Call(w.hwnd, 0, 0, 0x0001|0x0004|0x0080|0x0100)
}

func (w *window) command(id, notification int, source uintptr) {
	switch id {
	case idNavOverview:
		w.switchView(viewOverview)
	case idNavGames:
		w.switchView(viewGames)
	case idNavSettings:
		w.switchView(viewSettings)
	case idPauseNow:
		w.pauseNow()
	case idOpenLeiGod:
		w.openLeiGod()
	case idCheckAccount, idSettingsAccount:
		w.checkAccount()
	case idScanRunning:
		w.scanRunning()
	case idScanInstalled:
		w.scanInstalled()
	case idBrowseGame:
		w.browseGame()
	case idAddManual:
		w.addManualGame()
	case idToggleGame:
		w.toggleGame()
	case idRemoveGame:
		w.removeGame()
	case idCandidateAdd:
		w.addCandidate()
	case idCandidateBack:
		w.showCandidateMode(false)
	case idSaveSettings:
		w.saveSettings()
	case idClearToken:
		w.clearToken()
	case idClearPassword:
		w.clearPassword()
	case idBrowseLeiGod:
		w.browseLeiGod()
	case idFindLeiGod:
		w.findLeiGod()
	case idCandidateList:
		if notification == lbnDoubleClick {
			w.addCandidate()
		}
	}
}

func (w *window) updateOverview() {
	status, settings := w.monitor.Status(), w.store.Public()
	title := map[string]string{"idle": "等待游戏运行", "playing": "游戏运行中", "countdown": "即将暂停时长", "paused": "时长已暂停", "disabled": "监控已关闭", "error": "需要处理"}[status.Phase]
	if title == "" {
		title = "等待游戏运行"
	}
	message := status.Message
	if status.CountdownUntil != "" {
		if deadline, err := time.Parse(time.RFC3339, status.CountdownUntil); err == nil {
			remaining := int(time.Until(deadline).Seconds() + 0.99)
			if remaining < 0 {
				remaining = 0
			}
			message = fmt.Sprintf("%s（%d 秒）", status.Message, remaining)
		}
	}
	if status.LastError != "" {
		message += "：" + status.LastError
	}
	setText(w.statusTitle, title)
	setText(w.statusMessage, message)
	enabledGames := 0
	for _, game := range settings.Games {
		if game.Enabled {
			enabledGames++
		}
	}
	lastPause := "暂无"
	if status.LastPauseAt != "" {
		if stamp, err := time.Parse(time.RFC3339, status.LastPauseAt); err == nil {
			lastPause = stamp.Local().Format("01-02 15:04")
		}
	}
	monitorText := "关闭"
	if settings.Monitoring {
		monitorText = "开启"
	}
	setText(w.statusMetrics, fmt.Sprintf("监控：%s     游戏：%d 个     检测间隔：%d 秒     上次暂停：%s", monitorText, enabledGames, settings.CheckIntervalSec, lastPause))
	listReset(w.runningList)
	if len(status.RunningGames) == 0 {
		listAdd(w.runningList, "暂无监控中的游戏进程")
	} else {
		for _, name := range status.RunningGames {
			listAdd(w.runningList, "运行中  ·  "+name)
		}
	}
}

func (w *window) refreshGames() {
	games := w.store.Snapshot().Games
	listViewReset(w.gameList)
	for _, game := range games {
		state := "启用"
		if !game.Enabled {
			state = "停用"
		}
		source := game.Source
		if source == "" {
			source = "手动"
		}
		listViewAddRow(w.gameList, []string{state, game.Name, game.Executable, source})
	}
	if !w.candidateMode {
		w.updateCategorySummary()
	}
}

func (w *window) updateCategorySummary() {
	if w.categorySummary == 0 {
		return
	}
	if w.candidateMode {
		counts := map[string]int{}
		for _, item := range w.candidates {
			counts[candidateCategory(item)]++
		}
		setText(w.categorySummary, fmt.Sprintf("全部 %d  ·  游戏 %d  ·  平台 %d  ·  已安装 %d  ·  工具 %d  ·  其他 %d", len(w.candidates), counts["游戏"], counts["游戏平台"], counts["已安装游戏"], counts["应用工具"], counts["其他应用"]))
		return
	}
	settings := w.store.Snapshot()
	enabled := 0
	for _, game := range settings.Games {
		if game.Enabled {
			enabled++
		}
	}
	setText(w.categorySummary, fmt.Sprintf("监控列表  ·  全部 %d  ·  启用 %d  ·  停用 %d", len(settings.Games), enabled, len(settings.Games)-enabled))
}

func (w *window) showCandidateMode(show bool) {
	w.candidateMode = show
	if w.currentView != viewGames {
		showControls(w.gameHeaders, swHide)
		showControls(w.candidateHeaders, swHide)
		showWindow.Call(w.gameList, swHide)
		showWindow.Call(w.candidateList, swHide)
		showWindow.Call(w.candidateAdd, swHide)
		showWindow.Call(w.candidateBack, swHide)
		showWindow.Call(w.toggleGameButton, swHide)
		showWindow.Call(w.removeGameButton, swHide)
		return
	}
	if show {
		showControls(w.gameHeaders, swHide)
		showControls(w.candidateHeaders, swShow)
		showWindow.Call(w.gameList, swHide)
		showWindow.Call(w.toggleGameButton, swHide)
		showWindow.Call(w.removeGameButton, swHide)
		showWindow.Call(w.candidateList, swShow)
		showWindow.Call(w.candidateAdd, swShow)
		showWindow.Call(w.candidateBack, swShow)
		setText(w.gameSection, fmt.Sprintf("扫描结果（%d 个候选项）", len(w.candidates)))
		w.updateCategorySummary()
	} else {
		showControls(w.gameHeaders, swShow)
		showControls(w.candidateHeaders, swHide)
		showWindow.Call(w.gameList, swShow)
		showWindow.Call(w.toggleGameButton, swShow)
		showWindow.Call(w.removeGameButton, swShow)
		showWindow.Call(w.candidateList, swHide)
		showWindow.Call(w.candidateAdd, swHide)
		showWindow.Call(w.candidateBack, swHide)
		setText(w.gameSection, "已监控游戏")
		w.refreshGames()
	}
	redrawWindow.Call(w.hwnd, 0, 0, 0x0001|0x0004|0x0080|0x0100)
}

func (w *window) displayCandidates(items []config.Game) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := candidateCategory(items[i]), candidateCategory(items[j])
		if left == right {
			return strings.ToLower(items[i].Executable) < strings.ToLower(items[j].Executable)
		}
		return left < right
	})
	w.candidates = items
	listViewReset(w.candidateList)
	for _, item := range items {
		listViewAddRow(w.candidateList, []string{candidateCategory(item), item.Executable, item.Path, item.Source})
	}
	w.showCandidateMode(true)
}

func (w *window) scanRunning() {
	w.runAsync(func() asyncResult {
		items, err := w.scanner.List()
		if err != nil {
			return asyncResult{kind: "candidates", err: err}
		}
		seen := map[string]bool{}
		candidates := []config.Game{}
		for _, item := range items {
			name := strings.ToLower(item.Name)
			if item.Path == "" || !strings.HasSuffix(name, ".exe") || seen[name] || strings.Contains(strings.ToLower(item.Path), `\windows\`) || name == "leigodautopause.exe" || name == "leigod.exe" {
				continue
			}
			seen[name] = true
			candidates = append(candidates, config.Game{Name: strings.TrimSuffix(item.Name, filepath.Ext(item.Name)), Executable: item.Name, Path: item.Path, Source: "运行进程", Enabled: true})
		}
		return asyncResult{kind: "candidates", candidates: candidates}
	})
}

func (w *window) scanInstalled() {
	w.runAsync(func() asyncResult { return asyncResult{kind: "candidates", candidates: discovery.InstalledGames()} })
}

func candidateCategory(item config.Game) string {
	if strings.EqualFold(item.Source, "Steam") || strings.EqualFold(item.Source, "Epic") {
		return "已安装游戏"
	}
	lower := strings.ToLower(item.Executable + " " + item.Path)
	launcherName := strings.ToLower(item.Executable)
	launchers := []string{"steam.exe", "epicgameslauncher.exe", "battle.net.exe", "wegame.exe", "riotclientservices.exe", "ubisoftconnect.exe", "eadesktop.exe", "galaxyclient.exe"}
	for _, marker := range launchers {
		if launcherName == marker {
			return "游戏平台"
		}
	}
	if strings.Contains(launcherName, "launcher") {
		return "游戏平台"
	}
	if strings.Contains(lower, `\steamapps\`) || strings.Contains(lower, `\games\`) || strings.Contains(lower, `\game\`) {
		return "游戏"
	}
	browsers := []string{"chrome.exe", "msedge.exe", "firefox.exe", "brave.exe", "opera.exe"}
	for _, name := range browsers {
		if strings.EqualFold(item.Executable, name) {
			return "浏览器"
		}
	}
	tools := []string{"code.exe", "devenv.exe", "idea64.exe", "clion64.exe", "pycharm64.exe", "notepad.exe", "notepad++.exe", "wechat.exe", "qq.exe", "discord.exe"}
	for _, name := range tools {
		if strings.EqualFold(item.Executable, name) {
			return "应用工具"
		}
	}
	return "其他应用"
}

func (w *window) addCandidate() {
	index := listViewSelection(w.candidateList)
	if index < 0 || index >= len(w.candidates) {
		infoBox(w.hwnd, "请选择一个候选游戏。", mbIconWarning)
		return
	}
	if err := w.upsertGame(w.candidates[index]); err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
		return
	}
	infoBox(w.hwnd, "已添加到监控列表。", mbIconInfo)
	w.showCandidateMode(false)
}

func (w *window) browseGame() {
	path, ok := chooseExecutable(w.hwnd)
	if !ok {
		return
	}
	exe := filepath.Base(path)
	setText(w.gameExe, exe)
	if strings.TrimSpace(controlText(w.gameName)) == "" {
		setText(w.gameName, strings.TrimSuffix(exe, filepath.Ext(exe)))
	}
}

func (w *window) addManualGame() {
	name, executable := strings.TrimSpace(controlText(w.gameName)), filepath.Base(strings.TrimSpace(controlText(w.gameExe)))
	if executable == "" || !strings.HasSuffix(strings.ToLower(executable), ".exe") {
		infoBox(w.hwnd, "请输入有效的 .exe 进程名。", mbIconWarning)
		return
	}
	if name == "" {
		name = strings.TrimSuffix(executable, filepath.Ext(executable))
	}
	if err := w.upsertGame(config.Game{Name: name, Executable: executable, Source: "手动", Enabled: true}); err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
		return
	}
	setText(w.gameName, "")
	setText(w.gameExe, "")
	w.refreshGames()
	infoBox(w.hwnd, "游戏已添加。", mbIconInfo)
}

func (w *window) upsertGame(game config.Game) error {
	game.Executable = filepath.Base(strings.TrimSpace(game.Executable))
	if game.Name == "" || game.Executable == "" {
		return errors.New("游戏信息不完整")
	}
	game.Enabled = true
	err := w.store.Update(func(settings *config.Settings) error {
		for index := range settings.Games {
			if strings.EqualFold(settings.Games[index].Executable, game.Executable) {
				settings.Games[index] = game
				return nil
			}
		}
		settings.Games = append(settings.Games, game)
		return nil
	})
	if err == nil {
		w.monitor.Wake()
	}
	return err
}

func (w *window) toggleGame() {
	index := listViewSelection(w.gameList)
	games := w.store.Snapshot().Games
	if index < 0 || index >= len(games) {
		infoBox(w.hwnd, "请选择一个监控游戏。", mbIconWarning)
		return
	}
	exe := games[index].Executable
	err := w.store.Update(func(settings *config.Settings) error {
		for i := range settings.Games {
			if strings.EqualFold(settings.Games[i].Executable, exe) {
				settings.Games[i].Enabled = !settings.Games[i].Enabled
			}
		}
		return nil
	})
	if err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
		return
	}
	w.monitor.Wake()
	w.refreshGames()
}

func (w *window) removeGame() {
	index := listViewSelection(w.gameList)
	games := w.store.Snapshot().Games
	if index < 0 || index >= len(games) {
		infoBox(w.hwnd, "请选择一个监控游戏。", mbIconWarning)
		return
	}
	if questionBox(w.hwnd, "确定移除 “"+games[index].Name+"” 吗？") != idYes {
		return
	}
	exe := games[index].Executable
	_ = w.store.Update(func(settings *config.Settings) error {
		result := settings.Games[:0]
		for _, game := range settings.Games {
			if !strings.EqualFold(game.Executable, exe) {
				result = append(result, game)
			}
		}
		settings.Games = result
		return nil
	})
	w.monitor.Wake()
	w.refreshGames()
}

func (w *window) fillSettings() {
	settings := w.store.Public()
	w.setToggle(w.monitoring, settings.Monitoring)
	w.setToggle(w.autoPause, settings.AutoPause)
	w.setToggle(w.autoStart, settings.AutoStart)
	w.setToggle(w.silentStart, settings.SilentStart)
	w.setToggle(w.startMinimized, settings.StartMinimized)
	closeIndex := 0
	if settings.CloseAction == config.CloseTray {
		closeIndex = 1
	}
	if settings.CloseAction == config.CloseExit {
		closeIndex = 2
	}
	comboSelect(w.closeAction, closeIndex)
	setText(w.checkInterval, strconv.Itoa(settings.CheckIntervalSec))
	setText(w.gracePeriod, strconv.Itoa(settings.GracePeriodSec))
	setText(w.leiGodPath, settings.LeiGodPath)
	setText(w.username, settings.Username)
	setText(w.password, "")
	setText(w.token, "")
	passwordState, tokenState := "未保存", "未保存"
	if settings.HasPassword {
		passwordState = "已加密保存"
	}
	if settings.HasToken {
		tokenState = "已加密保存"
	}
	setText(w.credentialStatus, fmt.Sprintf("密码：%s     Token：%s", passwordState, tokenState))
}

func (w *window) saveSettings() {
	interval, err := strconv.Atoi(strings.TrimSpace(controlText(w.checkInterval)))
	if err != nil || interval < 2 || interval > 60 {
		infoBox(w.hwnd, "检测间隔必须为 2 到 60 秒。", mbIconWarning)
		return
	}
	grace, err := strconv.Atoi(strings.TrimSpace(controlText(w.gracePeriod)))
	if err != nil || grace < 0 || grace > 3600 {
		infoBox(w.hwnd, "宽限时间必须为 0 到 3600 秒。", mbIconWarning)
		return
	}
	err = w.store.Update(func(settings *config.Settings) error {
		settings.Monitoring, settings.AutoPause = w.toggleChecked(w.monitoring), w.toggleChecked(w.autoPause)
		settings.AutoStart, settings.SilentStart, settings.StartMinimized = w.toggleChecked(w.autoStart), w.toggleChecked(w.silentStart), w.toggleChecked(w.startMinimized)
		settings.CloseAction = []string{config.CloseAsk, config.CloseTray, config.CloseExit}[comboSelection(w.closeAction)]
		settings.CheckIntervalSec, settings.GracePeriodSec = interval, grace
		settings.LeiGodPath, settings.Username = strings.TrimSpace(controlText(w.leiGodPath)), strings.TrimSpace(controlText(w.username))
		return nil
	})
	if err == nil {
		err = startup.SetEnabled(w.toggleChecked(w.autoStart))
	}
	if err == nil && controlText(w.password) != "" {
		err = w.store.SetPasswordMD5(leigod.PasswordMD5(controlText(w.password)))
	}
	if err == nil && strings.TrimSpace(controlText(w.token)) != "" {
		err = w.store.SetToken(strings.TrimSpace(controlText(w.token)))
	}
	if err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
		return
	}
	w.monitor.Wake()
	w.fillSettings()
	infoBox(w.hwnd, "设置已保存。", mbIconInfo)
}

func (w *window) clearToken() {
	if questionBox(w.hwnd, "确定清除已保存的 Token 吗？") != idYes {
		return
	}
	if err := w.store.SetToken(""); err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
		return
	}
	w.fillSettings()
}

func (w *window) clearPassword() {
	if questionBox(w.hwnd, "确定清除已保存的密码摘要吗？") != idYes {
		return
	}
	if err := w.store.SetPasswordMD5(""); err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
		return
	}
	w.fillSettings()
}

func (w *window) pauseNow() {
	w.runAsync(func() asyncResult {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		message, err := w.client.Pause(ctx)
		w.monitor.RecordManualPause(message, err)
		return asyncResult{kind: "pause", message: message, err: err}
	})
}

func (w *window) pauseFromTray() {
	w.busyMu.Lock()
	if w.busy {
		w.busyMu.Unlock()
		w.showTrayNotification("雷神自动暂停", "另一项操作正在进行，请稍候。", true)
		return
	}
	w.busy = true
	w.busyMu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		message, err := w.client.Pause(ctx)
		w.monitor.RecordManualPause(message, err)
		w.results <- asyncResult{kind: "trayPause", message: message, err: err}
		postMessage.Call(w.hwnd, wmAppResult, 0, 0)
	}()
}

func (w *window) checkAccount() {
	w.runAsync(func() asyncResult {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		paused, message, err := w.client.AccountStatus(ctx)
		return asyncResult{kind: "account", paused: paused, message: message, err: err}
	})
}

func (w *window) openLeiGod() {
	path := resolveLeiGodPath(w.store.Snapshot().LeiGodPath, w.scanner)
	if path == "" {
		infoBox(w.hwnd, "未找到雷神加速器。请到“设置”点击“选择程序”，定位 leigod.exe。", mbIconWarning)
		return
	}
	_ = w.store.Update(func(settings *config.Settings) error { settings.LeiGodPath = path; return nil })
	setText(w.leiGodPath, path)
	verb, file, directory := utf16Ptr("open"), utf16Ptr(path), utf16Ptr(filepath.Dir(path))
	result, _, _ := shellExecute.Call(w.hwnd, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, uintptr(unsafe.Pointer(directory)), swShow)
	if result <= 32 {
		infoBox(w.hwnd, fmt.Sprintf("无法启动雷神加速器（ShellExecute 错误 %d）。请重新选择 leigod.exe。", result), mbIconError)
	}
}

func (w *window) browseLeiGod() {
	path, ok := chooseExecutable(w.hwnd)
	if !ok {
		return
	}
	if !strings.EqualFold(filepath.Base(path), "leigod.exe") {
		if questionBox(w.hwnd, "选择的程序不是 leigod.exe，仍然使用它吗？") != idYes {
			return
		}
	}
	setText(w.leiGodPath, path)
	if err := w.store.Update(func(settings *config.Settings) error { settings.LeiGodPath = path; return nil }); err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
	}
}

func (w *window) findLeiGod() {
	path := resolveLeiGodPath(controlText(w.leiGodPath), w.scanner)
	if path == "" {
		infoBox(w.hwnd, "未在常见安装目录或运行进程中找到 leigod.exe，请使用“选择程序”。", mbIconWarning)
		return
	}
	setText(w.leiGodPath, path)
	if err := w.store.Update(func(settings *config.Settings) error { settings.LeiGodPath = path; return nil }); err != nil {
		infoBox(w.hwnd, err.Error(), mbIconError)
		return
	}
	infoBox(w.hwnd, "已找到雷神加速器：\r\n"+path, mbIconInfo)
}

func resolveLeiGodPath(configured string, scanner processes.Scanner) string {
	configured = strings.Trim(strings.TrimSpace(configured), `"`)
	if isExecutableFile(configured) {
		return configured
	}
	if scanner != nil {
		if items, err := scanner.List(); err == nil {
			for _, item := range items {
				if strings.EqualFold(item.Name, "leigod.exe") && isExecutableFile(item.Path) {
					return item.Path
				}
			}
		}
	}
	bases := []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramData")}
	patterns := []string{`LeiGod_Acc\leigod.exe`, `LeiGod\leigod.exe`, `雷神加速器\leigod.exe`}
	for _, base := range bases {
		if base == "" {
			continue
		}
		for _, pattern := range patterns {
			if candidate := filepath.Join(base, pattern); isExecutableFile(candidate) {
				return candidate
			}
		}
		for _, pattern := range []string{"*LeiGod*", "*雷神*"} {
			matches, _ := filepath.Glob(filepath.Join(base, pattern, "leigod.exe"))
			for _, candidate := range matches {
				if isExecutableFile(candidate) {
					return candidate
				}
			}
		}
	}
	return ""
}

func isExecutableFile(path string) bool {
	if path == "" || !strings.EqualFold(filepath.Ext(path), ".exe") {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (w *window) runAsync(task func() asyncResult) {
	w.busyMu.Lock()
	if w.busy {
		w.busyMu.Unlock()
		infoBox(w.hwnd, "另一个操作正在进行，请稍候。", mbIconWarning)
		return
	}
	w.busy = true
	w.busyMu.Unlock()
	go func() { result := task(); w.results <- result; postMessage.Call(w.hwnd, wmAppResult, 0, 0) }()
}

func (w *window) handleResults() {
	for {
		select {
		case result := <-w.results:
			w.busyMu.Lock()
			w.busy = false
			w.busyMu.Unlock()
			if result.kind == "trayPause" {
				if result.err != nil {
					w.showTrayNotification("暂停失败", result.err.Error(), true)
				} else {
					message := strings.TrimSpace(result.message)
					if message == "" {
						message = "雷神加速时长已暂停。"
					}
					w.showTrayNotification("已立即暂停", message, false)
					w.updateOverview()
				}
				continue
			}
			if result.err != nil {
				infoBox(w.hwnd, result.err.Error(), mbIconError)
				continue
			}
			switch result.kind {
			case "candidates":
				w.displayCandidates(result.candidates)
			case "pause":
				infoBox(w.hwnd, result.message, mbIconInfo)
				w.updateOverview()
			case "account":
				text := result.message
				if result.paused {
					text += "\r\n当前不会消耗加速时长。"
				} else {
					text += "\r\n当前正在消耗加速时长。"
				}
				setText(w.accountText, text)
				infoBox(w.hwnd, text, mbIconInfo)
			}
		default:
			return
		}
	}
}

func (w *window) addTrayIcon() {
	if w.trayAdded || w.hwnd == 0 {
		return
	}
	data := notifyIconData{Size: uint32(unsafe.Sizeof(notifyIconData{})), Window: w.hwnd, ID: 1, Flags: nifMessage | nifIcon | nifTip, Callback: wmTray, Icon: w.icon}
	tip := utf16.Encode([]rune("雷神自动暂停"))
	copy(data.Tip[:], tip)
	if result, _, _ := shellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data))); result != 0 {
		w.trayAdded = true
	}
}

func (w *window) showTrayMenu() {
	menu, _, _ := createPopupMenu.Call()
	if menu == 0 {
		return
	}
	defer destroyMenu.Call(menu)
	appendMenu.Call(menu, mfString, idTrayShow, uintptr(unsafe.Pointer(utf16Ptr("显示窗口"))))
	appendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu.Call(menu, mfString, idTrayPause, uintptr(unsafe.Pointer(utf16Ptr("立即暂停"))))
	appendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu.Call(menu, mfString, idTrayExit, uintptr(unsafe.Pointer(utf16Ptr("退出"))))

	var cursor point
	if ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
		return
	}
	setForegroundWindow.Call(w.hwnd)
	command, _, _ := trackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton, uintptr(cursor.X), uintptr(cursor.Y), 0, w.hwnd, 0)
	postMessage.Call(w.hwnd, wmNull, 0, 0)
	switch command {
	case idTrayShow:
		w.removeTrayIcon()
		showWindow.Call(w.hwnd, swRestore)
		setForegroundWindow.Call(w.hwnd)
	case idTrayPause:
		w.pauseFromTray()
	case idTrayExit:
		destroyWindow.Call(w.hwnd)
	}
}

func (w *window) showTrayNotification(title, message string, isError bool) {
	if !w.trayAdded {
		return
	}
	data := notifyIconData{Size: uint32(unsafe.Sizeof(notifyIconData{})), Window: w.hwnd, ID: 1, Flags: nifInfo, InfoFlags: niifInfo}
	if isError {
		data.InfoFlags = niifError
	}
	copy(data.InfoTitle[:], utf16.Encode([]rune(title)))
	copy(data.Info[:], utf16.Encode([]rune(message)))
	shellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&data)))
}

func (w *window) removeTrayIcon() {
	if !w.trayAdded || w.hwnd == 0 {
		return
	}
	data := notifyIconData{Size: uint32(unsafe.Sizeof(notifyIconData{})), Window: w.hwnd, ID: 1}
	shellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	w.trayAdded = false
}

func (w *window) handleClose() {
	switch w.store.Snapshot().CloseAction {
	case config.CloseExit:
		destroyWindow.Call(w.hwnd)
	case config.CloseTray:
		w.addTrayIcon()
		showWindow.Call(w.hwnd, swHide)
	default:
		choice, _, _ := messageBox.Call(w.hwnd, uintptr(unsafe.Pointer(utf16Ptr("关闭程序？\r\n\r\n选择“是”退出程序，选择“否”隐藏到系统托盘。"))), uintptr(unsafe.Pointer(utf16Ptr(windowTitle))), mbYesNoCancel|mbIconWarning)
		if choice == idYes {
			destroyWindow.Call(w.hwnd)
		}
		if choice == idNo {
			w.addTrayIcon()
			showWindow.Call(w.hwnd, swHide)
		}
	}
}

func loadAppIcon() uintptr {
	directory := filepath.Join(os.Getenv("LOCALAPPDATA"), "LeiGodAutoPause")
	if directory == "LeiGodAutoPause" {
		if fallback, err := os.UserConfigDir(); err == nil {
			directory = filepath.Join(fallback, "LeiGodAutoPause")
		}
	}
	iconPath := filepath.Join(directory, "app.ico")
	if err := os.MkdirAll(directory, 0o700); err == nil {
		existing, readErr := os.ReadFile(iconPath)
		if readErr != nil || !bytes.Equal(existing, appIconData) {
			_ = os.WriteFile(iconPath, appIconData, 0o600)
		}
		if icon, _, _ := loadImage.Call(0, uintptr(unsafe.Pointer(utf16Ptr(iconPath))), imageIcon, 0, 0, lrLoadFromFile|lrDefaultSize); icon != 0 {
			return icon
		}
	}
	icon, _, _ := loadIcon.Call(0, idiApplication)
	return icon
}

func utf16Ptr(value string) *uint16 { pointer, _ := syscall.UTF16PtrFromString(value); return pointer }
func setText(hwnd uintptr, value string) {
	setWindowText.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(value))))
}
func controlText(hwnd uintptr) string {
	length, _, _ := getWindowTextLength.Call(hwnd)
	buffer := make([]uint16, length+1)
	getWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), length+1)
	return syscall.UTF16ToString(buffer)
}
func listReset(hwnd uintptr) { sendMessage.Call(hwnd, lbResetContent, 0, 0) }
func listAdd(hwnd uintptr, value string) {
	sendMessage.Call(hwnd, lbAddString, 0, uintptr(unsafe.Pointer(utf16Ptr(value))))
}
func listSelection(hwnd uintptr) int {
	result, _, _ := sendMessage.Call(hwnd, lbGetCurSel, 0, 0)
	if result == lbErr {
		return -1
	}
	return int(result)
}
func comboAdd(hwnd uintptr, value string) {
	sendMessage.Call(hwnd, cbAddString, 0, uintptr(unsafe.Pointer(utf16Ptr(value))))
}
func comboSelect(hwnd uintptr, index int) { sendMessage.Call(hwnd, cbSetCurSel, uintptr(index), 0) }
func comboSelection(hwnd uintptr) int {
	result, _, _ := sendMessage.Call(hwnd, cbGetCurSel, 0, 0)
	if int32(result) < 0 || result > 2 {
		return 0
	}
	return int(result)
}

type columnDef struct {
	Title string
	Width int32
}

func listViewColumns(hwnd uintptr, columns []columnDef) {
	sendMessage.Call(hwnd, lvmSetExtStyle, 0, lvexFullRow|lvexDoubleBuf)
	sendMessage.Call(hwnd, lvmSetBkColor, 0, rgb(249, 251, 253))
	sendMessage.Call(hwnd, lvmSetTextColor, 0, rgb(44, 56, 70))
	sendMessage.Call(hwnd, lvmSetTextBk, 0, rgb(249, 251, 253))
	for index, definition := range columns {
		text := utf16Ptr(definition.Title)
		column := listColumn{Mask: lvcfFmt | lvcfWidth | lvcfText | lvcfSubItem, Width: definition.Width, Text: text, SubItem: int32(index)}
		sendMessage.Call(hwnd, lvmInsertCol, uintptr(index), uintptr(unsafe.Pointer(&column)))
	}
}

func listViewReset(hwnd uintptr) { sendMessage.Call(hwnd, lvmDeleteAll, 0, 0) }

func listViewAddRow(hwnd uintptr, values []string) {
	if len(values) == 0 {
		return
	}
	first := utf16Ptr(values[0])
	item := listItem{Mask: lvifText, Item: 0x7fffffff, Text: first}
	row, _, _ := sendMessage.Call(hwnd, lvmInsertItem, 0, uintptr(unsafe.Pointer(&item)))
	if int32(row) < 0 {
		return
	}
	for index := 1; index < len(values); index++ {
		text := utf16Ptr(values[index])
		sub := listItem{SubItem: int32(index), Text: text}
		sendMessage.Call(hwnd, lvmSetItemText, row, uintptr(unsafe.Pointer(&sub)))
	}
}

func listViewSelection(hwnd uintptr) int {
	result, _, _ := sendMessage.Call(hwnd, lvmGetNext, ^uintptr(0), lvniSelected)
	if int32(result) < 0 {
		return -1
	}
	return int(result)
}
func (w *window) setToggle(hwnd uintptr, value bool) {
	if hwnd == 0 {
		return
	}
	w.toggleStates[hwnd] = value
	delete(w.toggleAnimation, hwnd)
	redrawWindow.Call(hwnd, 0, 0, 0x0001|0x0100)
}

func (w *window) toggleChecked(hwnd uintptr) bool {
	return w.toggleStates[hwnd]
}

func (w *window) startToggleAnimation(hwnd uintptr, current float64, now time.Time) {
	target := 0.0
	if w.toggleStates[hwnd] {
		target = 1
	}
	if !clientAnimationsEnabled() || current == target {
		delete(w.toggleAnimation, hwnd)
		redrawWindow.Call(hwnd, 0, 0, 0x0001|0x0100)
		return
	}
	w.toggleAnimation[hwnd] = toggleMotion{
		from:    current,
		to:      target,
		started: now,
		dur:     toggleAnimationMillis * time.Millisecond,
	}
	if !w.toggleTimerActive {
		timeBeginPeriod.Call(1)
		w.toggleTimerActive = true
		setTimer.Call(w.hwnd, 2, toggleFrameMillis, 0)
	}
	redrawWindow.Call(hwnd, 0, 0, 0x0001|0x0100)
}

func (w *window) toggleProgress(hwnd uintptr, now time.Time) float64 {
	motion, ok := w.toggleAnimation[hwnd]
	if !ok {
		if w.toggleStates[hwnd] {
			return 1
		}
		return 0
	}
	if motion.dur <= 0 {
		return motion.to
	}
	elapsed := now.Sub(motion.started)
	if elapsed <= 0 {
		return motion.from
	}
	if elapsed >= motion.dur {
		return motion.to
	}
	ratio := float64(elapsed) / float64(motion.dur)
	eased := ratio * ratio * (3 - 2*ratio)
	return motion.from + (motion.to-motion.from)*eased
}

func (w *window) advanceToggleAnimations() {
	now := time.Now()
	for hwnd, motion := range w.toggleAnimation {
		if now.Sub(motion.started) >= motion.dur {
			delete(w.toggleAnimation, hwnd)
		}
		redrawWindow.Call(hwnd, 0, 0, 0x0001|0x0100)
	}
	if len(w.toggleAnimation) == 0 {
		w.stopToggleTimer()
	}
}

func (w *window) stopToggleTimer() {
	if !w.toggleTimerActive {
		return
	}
	killTimer.Call(w.hwnd, 2)
	timeEndPeriod.Call(1)
	w.toggleTimerActive = false
}

func clientAnimationsEnabled() bool {
	enabled := int32(1)
	result, _, _ := systemParametersInfo.Call(0x1042, 0, uintptr(unsafe.Pointer(&enabled)), 0)
	return result == 0 || enabled != 0
}

func infoBox(owner uintptr, text string, icon uintptr) {
	messageBox.Call(owner, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(windowTitle))), mbOK|icon)
}
func questionBox(owner uintptr, text string) uintptr {
	result, _, _ := messageBox.Call(owner, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(windowTitle))), mbYesNo|mbIconWarning)
	return result
}

func chooseExecutable(owner uintptr) (string, bool) {
	buffer := make([]uint16, 32768)
	filterText := "可执行文件 (*.exe)\x00*.exe\x00所有文件 (*.*)\x00*.*\x00\x00"
	filter := append(utf16.Encode([]rune(filterText)), 0)
	title := utf16Ptr("选择游戏可执行文件")
	dialog := openFileName{Size: uint32(unsafe.Sizeof(openFileName{})), Owner: owner, Filter: &filter[0], FilterIndex: 1, File: &buffer[0], MaxFile: uint32(len(buffer)), Title: title, Flags: 0x00080000 | 0x00001000 | 0x00000800}
	ok, _, _ := getOpenFileName.Call(uintptr(unsafe.Pointer(&dialog)))
	if ok == 0 {
		return "", false
	}
	return syscall.UTF16ToString(buffer), true
}

//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	shell_NotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	shellExecuteW     = shell32.NewProc("ShellExecuteW")
	extractIconExW    = shell32.NewProc("ExtractIconExW")

	registerClassExW     = user32.NewProc("RegisterClassExW")
	createWindowExW_tray = user32.NewProc("CreateWindowExW")
	getMessageW          = user32.NewProc("GetMessageW")
	translateMessage     = user32.NewProc("TranslateMessage")
	dispatchMessageW     = user32.NewProc("DispatchMessageW")
	postQuitMessage      = user32.NewProc("PostQuitMessage")
	defWindowProcW       = user32.NewProc("DefWindowProcW")
	createPopupMenu      = user32.NewProc("CreatePopupMenu")
	appendMenuW          = user32.NewProc("AppendMenuW")
	trackPopupMenu       = user32.NewProc("TrackPopupMenu")
	destroyMenu          = user32.NewProc("DestroyMenu")
	getCursorPos         = user32.NewProc("GetCursorPos")
	setForegroundWnd     = user32.NewProc("SetForegroundWindow")
	loadImageW           = user32.NewProc("LoadImageW")
	loadIconW            = user32.NewProc("LoadIconW")

	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

const (
	NIM_ADD    = 0x00000000
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	WM_APP       = 0x8000
	WM_TRAYICON  = WM_APP + 1
	WM_COMMAND   = 0x0111
	WM_DESTROY   = 0x0002
	WM_RBUTTONUP = 0x0205
	WM_LBUTTONUP = 0x0202

	MF_STRING       = 0x00000000
	MF_SEPARATOR    = 0x00000800
	MF_GRAYED       = 0x00000001
	TPM_BOTTOMALIGN = 0x0020
	TPM_LEFTALIGN   = 0x0000

	IDM_OPEN = 1001
	IDM_QUIT = 1002

	IMAGE_ICON       = 1
	LR_LOADFROMFILE  = 0x00000010
	WS_EX_TOOLWINDOW = 0x00000080
	IDI_APPLICATION  = 32512
)

// NOTIFYICONDATAW_V1 — matches Windows NOTIFYICONDATAW original version (168 bytes on 64-bit)
// Go adds correct padding automatically between uint32 and uintptr fields
type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [64]uint16
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm      uintptr
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X, Y int32
}

var (
	trayHwnd uintptr
	trayNid  NOTIFYICONDATAW
)

func initSystray() {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hInstance, _, _ := getModuleHandleW.Call(0)
		className := utf16PtrFromString("CZManagerTray")

		wc := WNDCLASSEX{
			CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
			LpfnWndProc:   syscall.NewCallback(trayWndProc),
			HInstance:     hInstance,
			LpszClassName: className,
		}
		registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))

		trayHwnd, _, _ = createWindowExW_tray.Call(
			WS_EX_TOOLWINDOW,
			uintptr(unsafe.Pointer(className)),
			0, 0,
			0, 0, 0, 0,
			0, 0, hInstance, 0,
		)
		if trayHwnd == 0 {
			fmt.Println("Systray: failed to create window")
			return
		}

		hIcon := loadTrayIcon()
		if hIcon == 0 {
			hIcon, _, _ = loadIconW.Call(0, IDI_APPLICATION)
		}
		if hIcon == 0 {
			fmt.Println("Systray: no icon available")
			return
		}

		trayNid = NOTIFYICONDATAW{
			CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
			HWnd:             trayHwnd,
			UID:              1,
			UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
			UCallbackMessage: WM_TRAYICON,
			HIcon:            hIcon,
		}

		tip := fmt.Sprintf("CZManager Agent v%s", Version)
		tipUtf16, _ := syscall.UTF16FromString(tip)
		n := copy(trayNid.SzTip[:], tipUtf16)
		if n < len(trayNid.SzTip) {
			trayNid.SzTip[n] = 0
		}

		ret, _, _ := shell_NotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&trayNid)))
		if ret == 0 {
			fmt.Println("Systray: Shell_NotifyIconW NIM_ADD failed")
			return
		}

		var msg MSG
		for {
			ret, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if ret == 0 {
				break
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()
}

func removeSystray() {
	if trayHwnd != 0 {
		shell_NotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&trayNid)))
	}
}

func loadTrayIcon() uintptr {
	// Write embedded favicon to temp file and load it
	if len(faviconICO) > 0 {
		tmpFile := filepath.Join(os.TempDir(), "czmanager-tray.ico")
		if err := os.WriteFile(tmpFile, faviconICO, 0644); err == nil {
			pathPtr, _ := syscall.UTF16PtrFromString(tmpFile)
			icon, _, _ := loadImageW.Call(0, uintptr(unsafe.Pointer(pathPtr)), IMAGE_ICON, 16, 16, LR_LOADFROMFILE)
			if icon != 0 {
				return icon
			}
		}
	}

	// Fallback: extract icon from shell32.dll
	shell32Path, _ := syscall.UTF16PtrFromString(`C:\Windows\System32\shell32.dll`)
	var smallIcon uintptr
	ret, _, _ := extractIconExW.Call(uintptr(unsafe.Pointer(shell32Path)), 2, 0, uintptr(unsafe.Pointer(&smallIcon)), 1)
	if ret > 0 && smallIcon != 0 {
		return smallIcon
	}

	return 0
}

func trayWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_RBUTTONUP, WM_LBUTTONUP:
			showTrayMenu(hwnd)
		}
		return 0
	case WM_COMMAND:
		switch wParam {
		case IDM_OPEN:
			openBrowser("https://lokalizace.net")
		case IDM_QUIT:
			removeSystray()
			os.Exit(0)
		}
		return 0
	case WM_DESTROY:
		removeSystray()
		postQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := defWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func showTrayMenu(hwnd uintptr) {
	hMenu, _, _ := createPopupMenu.Call()
	if hMenu == 0 {
		return
	}
	statusText := fmt.Sprintf("CZManager Agent v%s", Version)
	appendMenuW.Call(hMenu, MF_STRING|MF_GRAYED, 0, uintptr(unsafe.Pointer(utf16PtrFromString(statusText))))
	appendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	appendMenuW.Call(hMenu, MF_STRING, IDM_OPEN, uintptr(unsafe.Pointer(utf16PtrFromString("Otevřít lokalizace.net"))))
	appendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	appendMenuW.Call(hMenu, MF_STRING, IDM_QUIT, uintptr(unsafe.Pointer(utf16PtrFromString("Ukončit"))))

	var pt POINT
	getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	setForegroundWnd.Call(hwnd)
	trackPopupMenu.Call(hMenu, TPM_BOTTOMALIGN|TPM_LEFTALIGN, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	destroyMenu.Call(hMenu)
}

func openBrowser(url string) {
	shellExecuteW.Call(0, uintptr(unsafe.Pointer(utf16PtrFromString("open"))), uintptr(unsafe.Pointer(utf16PtrFromString(url))), 0, 0, 1)
}

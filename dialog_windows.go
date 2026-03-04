//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	ole32            = syscall.NewLazyDLL("ole32.dll")
	comdlg32         = syscall.NewLazyDLL("comdlg32.dll")
	shBrowseForFolder = shell32.NewProc("SHBrowseForFolderW")
	shGetPathFromIDList = shell32.NewProc("SHGetPathFromIDListW")
	coInitializeEx   = ole32.NewProc("CoInitializeEx")
	coUninitialize   = ole32.NewProc("CoUninitialize")
	getOpenFileName  = comdlg32.NewProc("GetOpenFileNameW")
	getForegroundWindow = user32.NewProc("GetForegroundWindow")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
)

const (
	BIF_RETURNONLYFSDIRS = 0x00000001
	BIF_NEWDIALOGSTYLE   = 0x00000040
	BFFM_SETSELECTION    = 0x00000467
	MAX_PATH             = 260
	OFN_FILEMUSTEXIST    = 0x00001000
	OFN_PATHMUSTEXIST    = 0x00000800
	OFN_NOCHANGEDIR      = 0x00000008
	COINIT_APARTMENTTHREADED = 0x2
)

type BROWSEINFO struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

type OPENFILENAME struct {
	LStructSize       uint32
	HwndOwner         uintptr
	HInstance         uintptr
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        uintptr
	DwReserved        uint32
	FlagsEx           uint32
}

func utf16PtrFromString(s string) *uint16 {
	ptr, _ := syscall.UTF16PtrFromString(s)
	return ptr
}

func browseForFolder(title string) (string, bool, error) {
	// Initialize COM
	coInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	defer coUninitialize.Call()

	// Show toast notification to alert user
	showToastNotification("CZManager Agent", "Vyberte složku s hrou")

	// Bring our process to foreground before opening dialog
	bringToForeground()

	displayName := make([]uint16, MAX_PATH)

	bi := BROWSEINFO{
		HwndOwner:      0,
		PidlRoot:       0,
		PszDisplayName: &displayName[0],
		LpszTitle:      utf16PtrFromString(title),
		UlFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE,
		Lpfn:           0,
		LParam:         0,
	}

	pidl, _, _ := shBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", true, nil // User cancelled
	}

	path := make([]uint16, MAX_PATH)
	shGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&path[0])))

	return syscall.UTF16ToString(path), false, nil
}

func browseForFile(title, filter, startPath string) (string, bool, error) {
	// Initialize COM
	coInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	defer coUninitialize.Call()

	// Bring our process to foreground before opening dialog
	bringToForeground()

	fileBuffer := make([]uint16, MAX_PATH*2)

	// Build filter string (format: "Description\0*.ext\0\0")
	var filterStr *uint16
	if filter != "" {
		// Simple filter: "All Files\0*.*\0\0"
		filterBytes := make([]uint16, 256)
		copy(filterBytes, utf16FromString("All Files"))
		filterBytes[9] = 0
		copy(filterBytes[10:], utf16FromString("*.*"))
		filterBytes[14] = 0
		filterBytes[15] = 0
		filterStr = &filterBytes[0]
	}

	var initialDir *uint16
	if startPath != "" {
		initialDir = utf16PtrFromString(startPath)
	}

	ofn := OPENFILENAME{
		LStructSize:     uint32(unsafe.Sizeof(OPENFILENAME{})),
		HwndOwner:       0,
		LpstrFilter:     filterStr,
		LpstrFile:       &fileBuffer[0],
		NMaxFile:        MAX_PATH * 2,
		LpstrTitle:      utf16PtrFromString(title),
		LpstrInitialDir: initialDir,
		Flags:           OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR,
	}

	ret, _, _ := getOpenFileName.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return "", true, nil // User cancelled
	}

	return syscall.UTF16ToString(fileBuffer), false, nil
}

func utf16FromString(s string) []uint16 {
	result, _ := syscall.UTF16FromString(s)
	return result
}

var (
	createWindowExW    = user32.NewProc("CreateWindowExW")
	destroyWindow      = user32.NewProc("DestroyWindow")
	setWindowPos       = user32.NewProc("SetWindowPos")
	attachThreadInput  = user32.NewProc("AttachThreadInput")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	getCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
	allowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
)

const (
	HWND_TOPMOST   = ^uintptr(0) // -1
	SWP_NOMOVE     = 0x0002
	SWP_NOSIZE     = 0x0001
	SWP_SHOWWINDOW = 0x0040
	ASFW_ANY       = 0xFFFFFFFF
)

// bringToForeground attempts to bring dialogs to foreground
func bringToForeground() {
	// Allow any process to set foreground window
	allowSetForegroundWindow.Call(ASFW_ANY)

	// Get current foreground window's thread
	fgHwnd, _, _ := getForegroundWindow.Call()
	if fgHwnd == 0 {
		return
	}

	fgThread, _, _ := getWindowThreadProcessId.Call(fgHwnd, 0)
	ourThread, _, _ := getCurrentThreadId.Call()

	// Attach our thread input to foreground thread to steal focus
	if fgThread != ourThread {
		attachThreadInput.Call(ourThread, fgThread, 1) // Attach
		setForegroundWindow.Call(fgHwnd)
		attachThreadInput.Call(ourThread, fgThread, 0) // Detach
	}
}

// showToastNotification shows a Windows toast notification
func showToastNotification(title, message string) {
	// Use PowerShell to show toast notification (works on Windows 10+)
	script := `
	[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
	[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
	$template = @"
	<toast>
		<visual>
			<binding template="ToastText02">
				<text id="1">` + title + `</text>
				<text id="2">` + message + `</text>
			</binding>
		</visual>
		<audio silent="true"/>
	</toast>
"@
	$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
	$xml.LoadXml($template)
	$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
	[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("CZManager Agent").Show($toast)
	`

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.Start() // Fire and forget - don't wait for completion
}

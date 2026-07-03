//go:build windows

package scanner

import (
	"syscall"
	"unsafe"
)

// getAvailableDrives returns all available drive letters on Windows
func getAvailableDrives() []string {
	var drives []string

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getLogicalDrives := kernel32.NewProc("GetLogicalDrives")

	ret, _, _ := getLogicalDrives.Call()
	bitMask := uint32(ret)

	for i := 0; i < 26; i++ {
		if bitMask&(1<<uint(i)) != 0 {
			driveLetter := string(rune('A'+i)) + ":\\"
			// Check if drive is accessible (skip CD-ROMs, etc.)
			driveType := getDriveType(driveLetter)
			if driveType == 3 { // DRIVE_FIXED
				drives = append(drives, driveLetter)
			}
		}
	}

	return drives
}

// getDriveType returns the type of the specified drive
func getDriveType(rootPath string) uint32 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDriveTypeW := kernel32.NewProc("GetDriveTypeW")

	rootPathPtr, _ := syscall.UTF16PtrFromString(rootPath)
	ret, _, _ := getDriveTypeW.Call(uintptr(unsafe.Pointer(rootPathPtr)))
	return uint32(ret)
}

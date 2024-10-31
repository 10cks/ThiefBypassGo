// icon_changer.go

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	GENERIC_READ  = 0x80000000
	OPEN_EXISTING = 3
	FILE_BEGIN    = 0
	RT_ICON       = 3
	RT_GROUP_ICON = 14
)

func changeExecutableIcon(iconPath string, executablePath string) bool {
	// Check if files exist
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		fmt.Println("Icon not found!")
		return false
	}

	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		fmt.Println("Executable not found!")
		return false
	}

	// Open icon file
	iconFile, err := syscall.CreateFile(
		syscall.StringToUTF16Ptr(iconPath),
		GENERIC_READ,
		0,
		nil,
		OPEN_EXISTING,
		0,
		0)
	if err != nil {
		fmt.Printf("Failed to open icon file: %v\n", err)
		return false
	}
	defer syscall.CloseHandle(iconFile)

	// Read icon header (first 22 bytes)
	iconInfo := make([]byte, 22)
	var bytesRead uint32
	err = syscall.ReadFile(iconFile, iconInfo, &bytesRead, nil)
	if err != nil {
		fmt.Printf("Failed to read icon header: %v\n", err)
		return false
	}

	// Validate icon format
	if binary.LittleEndian.Uint16(iconInfo[0:2]) != 0 || binary.LittleEndian.Uint16(iconInfo[2:4]) != 1 {
		fmt.Println("Icon is not a valid .ico file!")
		return false
	}

	// Get image offset and size
	imageOffset := binary.LittleEndian.Uint32(iconInfo[18:22])
	imageSize := binary.LittleEndian.Uint32(iconInfo[14:18])

	// Read icon image data
	iconData := make([]byte, imageSize)
	_, err = syscall.Seek(iconFile, int64(imageOffset), FILE_BEGIN)
	if err != nil {
		fmt.Printf("Failed to seek icon file: %v\n", err)
		return false
	}

	err = syscall.ReadFile(iconFile, iconData, &bytesRead, nil)
	if err != nil {
		fmt.Printf("Failed to read icon data: %v\n", err)
		return false
	}

	// Modify icon info
	binary.LittleEndian.PutUint16(iconInfo[4:6], 1)
	binary.LittleEndian.PutUint16(iconInfo[18:20], 1)

	// Update executable resources
	h, err := BeginUpdateResource(executablePath, false)
	if err != nil {
		fmt.Printf("Failed to begin update resource: %v\n", err)
		return false
	}

	err = UpdateResource(h, RT_ICON, 1, 0, unsafe.Pointer(&iconData[0]), uint32(len(iconData)))
	if err != nil {
		fmt.Printf("Failed to update icon resource: %v\n", err)
		return false
	}

	err = UpdateResource(h, RT_GROUP_ICON, 1, 0, unsafe.Pointer(&iconInfo[0]), 20)
	if err != nil {
		fmt.Printf("Failed to update group icon resource: %v\n", err)
		return false
	}

	err = EndUpdateResource(h, false)
	if err != nil {
		fmt.Printf("Failed to end update resource: %v\n", err)
		return false
	}

	return true
}

// Windows API functions
var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	beginUpdateResourceW = kernel32.NewProc("BeginUpdateResourceW")
	updateResourceW      = kernel32.NewProc("UpdateResourceW")
	endUpdateResourceW   = kernel32.NewProc("EndUpdateResourceW")
)

func BeginUpdateResource(fileName string, deleteExistingResources bool) (syscall.Handle, error) {
	ret, _, err := beginUpdateResourceW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(fileName))),
		uintptr(boolToInt(deleteExistingResources)))

	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func UpdateResource(handle syscall.Handle, resourceType uintptr, resourceName uintptr,
	languageID uint16, data unsafe.Pointer, size uint32) error {
	ret, _, err := updateResourceW.Call(
		uintptr(handle),
		resourceType,
		resourceName,
		uintptr(languageID),
		uintptr(data),
		uintptr(size))

	if ret == 0 {
		return err
	}
	return nil
}

func EndUpdateResource(handle syscall.Handle, discard bool) error {
	ret, _, err := endUpdateResourceW.Call(
		uintptr(handle),
		uintptr(boolToInt(discard)))

	if ret == 0 {
		return err
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("Icon-Changer version 1.0.0!\n")
		os.Exit(0)
	}

	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <path_to_icon> <path_to_exe>\n", os.Args[0])
		os.Exit(1)
	}

	if changeExecutableIcon(os.Args[1], os.Args[2]) {
		os.Exit(0)
	}
	os.Exit(1)
}

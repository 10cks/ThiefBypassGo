package main

import (
	"encoding/binary"
	"fmt"
	"github.com/orcastor/fico"
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

// Windows API functions
var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	beginUpdateResourceW = kernel32.NewProc("BeginUpdateResourceW")
	updateResourceW      = kernel32.NewProc("UpdateResourceW")
	endUpdateResourceW   = kernel32.NewProc("EndUpdateResourceW")
)

// ConvertToICO 将输入文件转换为ICO格式
func ConvertToICO(inputPath, outputPath string, width, height int, iconIndex *int) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	cfg := fico.Config{
		Format: "ico",
		Width:  width,
		Height: height,
		Index:  iconIndex,
	}

	return fico.F2ICO(outFile, inputPath, cfg)
}

func changeExecutableIcon(iconPath string, executablePath string) bool {
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		fmt.Println("Icon not found!")
		return false
	}

	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		fmt.Println("Executable not found!")
		return false
	}

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

	iconInfo := make([]byte, 22)
	var bytesRead uint32
	err = syscall.ReadFile(iconFile, iconInfo, &bytesRead, nil)
	if err != nil {
		fmt.Printf("Failed to read icon header: %v\n", err)
		return false
	}

	if binary.LittleEndian.Uint16(iconInfo[0:2]) != 0 || binary.LittleEndian.Uint16(iconInfo[2:4]) != 1 {
		fmt.Println("Icon is not a valid .ico file!")
		return false
	}

	imageOffset := binary.LittleEndian.Uint32(iconInfo[18:22])
	imageSize := binary.LittleEndian.Uint32(iconInfo[14:18])

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

	binary.LittleEndian.PutUint16(iconInfo[4:6], 1)
	binary.LittleEndian.PutUint16(iconInfo[18:20], 1)

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

func printUsage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("  Extract icon:    %s extract <input_file> <output_ico> [width] [height] [icon_index]\n", os.Args[0])
	fmt.Printf("  Change icon:     %s change <ico_file> <exe_file>\n", os.Args[0])
	fmt.Printf("  Version:         %s --version|-v\n", os.Args[0])
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--version", "-v":
		fmt.Printf("Icon Tool version 1.0.0\n")
		os.Exit(0)

	case "extract":
		if len(os.Args) < 4 {
			printUsage()
			os.Exit(1)
		}

		width := 0
		height := 0
		var iconIndex *int

		if len(os.Args) > 4 {
			fmt.Sscanf(os.Args[4], "%d", &width)
		}
		if len(os.Args) > 5 {
			fmt.Sscanf(os.Args[5], "%d", &height)
		}
		if len(os.Args) > 6 {
			var idx int
			fmt.Sscanf(os.Args[6], "%d", &idx)
			iconIndex = &idx
		}

		err := ConvertToICO(os.Args[2], os.Args[3], width, height, iconIndex)
		if err != nil {
			fmt.Printf("Extract failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Extract success!")

	case "change":
		if len(os.Args) != 4 {
			printUsage()
			os.Exit(1)
		}

		if changeExecutableIcon(os.Args[2], os.Args[3]) {
			fmt.Println("Change icon success!")
			os.Exit(0)
		}
		fmt.Println("Change icon failed!")
		os.Exit(1)

	default:
		printUsage()
		os.Exit(1)
	}
}

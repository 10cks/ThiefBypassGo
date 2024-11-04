package main

import (
	"encoding/binary"
	"flag"
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

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	beginUpdateResourceW = kernel32.NewProc("BeginUpdateResourceW")
	updateResourceW      = kernel32.NewProc("UpdateResourceW")
	endUpdateResourceW   = kernel32.NewProc("EndUpdateResourceW")
)

type commandFlags struct {
	version    bool
	mode       string
	inputFile  string
	outputFile string
	width      int
	height     int
	iconIndex  int
	useIndex   bool
	iconFile   string
	exeFile    string
}

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

func main() {
	flags := commandFlags{}

	// 设置通用标志
	flag.StringVar(&flags.mode, "mode", "", "操作模式: extract 或 change")
	flag.BoolVar(&flags.version, "version", false, "显示版本信息")
	flag.BoolVar(&flags.version, "v", false, "显示版本信息（简写）")

	// extract 模式的标志
	flag.StringVar(&flags.inputFile, "input", "", "输入文件路径")
	flag.StringVar(&flags.outputFile, "output", "", "输出ICO文件路径")
	flag.IntVar(&flags.width, "width", 0, "图标宽度")
	flag.IntVar(&flags.height, "height", 0, "图标高度")
	flag.IntVar(&flags.iconIndex, "index", 0, "图标索引")
	flag.BoolVar(&flags.useIndex, "use-index", false, "是否使用图标索引")

	// change 模式的标志
	flag.StringVar(&flags.iconFile, "icon", "", "ICO文件路径")
	flag.StringVar(&flags.exeFile, "exe", "", "可执行文件路径")

	flag.Parse()

	if flags.version {
		fmt.Printf("Icon Tool version 1.0.0\n")
		os.Exit(0)
	}

	switch flags.mode {
	case "icon-extract":
		if flags.inputFile == "" || flags.outputFile == "" {
			fmt.Println("Usage: program -mode extract -input <input_file> -output <output_ico> [-width width] [-height height] [-index icon_index]")
			os.Exit(1)
		}
		var iconIndex *int
		if flags.useIndex {
			iconIndex = &flags.iconIndex
		}
		err := ConvertToICO(flags.inputFile, flags.outputFile, flags.width, flags.height, iconIndex)
		if err != nil {
			fmt.Printf("Extract failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Extract success!")

	case "icon-change":
		if flags.iconFile == "" || flags.exeFile == "" {
			fmt.Println("Usage: program -mode change -icon <ico_file> -exe <exe_file>")
			os.Exit(1)
		}
		if changeExecutableIcon(flags.iconFile, flags.exeFile) {
			fmt.Println("Change icon success!")
			os.Exit(0)
		}
		fmt.Println("Change icon failed!")
		os.Exit(1)

	default:
		fmt.Println("Usage:")
		fmt.Println("  Extract: program -mode icon-extract -input <input_file> -output <output_ico> [-width width] [-height height] [-index icon_index]")
		fmt.Println("  Change:  program -mode icon-change -icon <ico_file> -exe <exe_file>")
		fmt.Println("  Version: program -version|-v")
		os.Exit(1)
	}
}

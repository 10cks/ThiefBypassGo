package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/orcastor/fico"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"unsafe"
)

const (
	GENERIC_READ             = 0x80000000
	OPEN_EXISTING            = 3
	FILE_BEGIN               = 0
	RT_ICON                  = 3
	RT_GROUP_ICON            = 14
	LOAD_LIBRARY_AS_DATAFILE = 0x2
	LANG_NEUTRAL             = 0x0
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	beginUpdateResourceW = kernel32.NewProc("BeginUpdateResourceW")
	updateResourceW      = kernel32.NewProc("UpdateResourceW")
	endUpdateResourceW   = kernel32.NewProc("EndUpdateResourceW")
	procLoadLibraryEx    = kernel32.NewProc("LoadLibraryExW")
	procFindResourceEx   = kernel32.NewProc("FindResourceExW")
	procLoadResource     = kernel32.NewProc("LoadResource")
	procLockResource     = kernel32.NewProc("LockResource")
	procSizeofResource   = kernel32.NewProc("SizeofResource")
	procFreeLibrary      = kernel32.NewProc("FreeLibrary")
)

type commandFlags struct {
	version      bool
	mode         string
	inputFile    string
	outputFile   string
	width        int
	height       int
	iconIndex    int
	useIndex     bool
	iconFile     string
	exeFile      string
	resourceType uint16
	resourceID   uint16
	languageID   uint16
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

func AddResourceFromFile(peFile string, resFile string, resType uint16, resName uint16, resLang uint16) error {
	data, err := ioutil.ReadFile(resFile)
	if err != nil {
		return fmt.Errorf("read resource file failed: %v", err)
	}

	peFileW, err := syscall.UTF16PtrFromString(peFile)
	if err != nil {
		return fmt.Errorf("convert filename failed: %v", err)
	}

	handle, _, err := beginUpdateResourceW.Call(
		uintptr(unsafe.Pointer(peFileW)),
		uintptr(0))

	if handle == 0 {
		return fmt.Errorf("BeginUpdateResource failed: %v", err)
	}

	success, _, err := updateResourceW.Call(
		handle,
		uintptr(resType),
		uintptr(resName),
		uintptr(resLang),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)))

	if success == 0 {
		return fmt.Errorf("UpdateResource failed: %v", err)
	}

	success, _, err = endUpdateResourceW.Call(
		handle,
		uintptr(0))

	if success == 0 {
		return fmt.Errorf("EndUpdateResource failed: %v", err)
	}

	return nil
}

func ExtractResourceToFile(peFile string, outFile string, resType uint16, resName uint16, resLang uint16) error {
	peFileW, err := syscall.UTF16PtrFromString(peFile)
	if err != nil {
		return fmt.Errorf("convert filename failed: %v", err)
	}

	hModule, _, err := procLoadLibraryEx.Call(
		uintptr(unsafe.Pointer(peFileW)),
		0,
		LOAD_LIBRARY_AS_DATAFILE)

	if hModule == 0 {
		return fmt.Errorf("LoadLibraryEx failed: %v", err)
	}
	defer procFreeLibrary.Call(hModule)

	hResInfo, _, err := procFindResourceEx.Call(
		hModule,
		uintptr(resType),
		uintptr(resName),
		uintptr(resLang))

	if hResInfo == 0 {
		return fmt.Errorf("FindResourceEx failed: %v", err)
	}

	hResData, _, err := procLoadResource.Call(hModule, hResInfo)
	if hResData == 0 {
		return fmt.Errorf("LoadResource failed: %v", err)
	}

	lpResData, _, err := procLockResource.Call(hResData)
	if lpResData == 0 {
		return fmt.Errorf("LockResource failed: %v", err)
	}

	size, _, err := procSizeofResource.Call(hModule, hResInfo)
	if size == 0 {
		return fmt.Errorf("SizeofResource failed: %v", err)
	}

	data := make([]byte, size)
	copy(data, unsafe.Slice((*byte)(unsafe.Pointer(lpResData)), size))

	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return fmt.Errorf("create output directory failed: %v", err)
	}

	if err := ioutil.WriteFile(outFile, data, 0644); err != nil {
		return fmt.Errorf("write output file failed: %v", err)
	}

	return nil
}

func ReplaceResource(inputFile, outputFile string, resType, resID, resLang uint16) error {
	err := ExtractResourceToFile(inputFile, "temp_resource", resType, resID, resLang)
	if err != nil {
		return fmt.Errorf("extract resource failed: %v", err)
	}

	err = AddResourceFromFile(outputFile, "temp_resource", resType, resID, resLang)
	if err != nil {
		return fmt.Errorf("add resource failed: %v", err)
	}

	os.Remove("temp_resource")
	return nil
}

func ReplaceIcon(inputFile, outputFile string) error {
	err := ConvertToICO(inputFile, "temp_icon.ico", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("convert to ICO failed: %v", err)
	}

	success := changeExecutableIcon("temp_icon.ico", outputFile)
	os.Remove("temp_icon.ico")

	if !success {
		return fmt.Errorf("replace icon failed")
	}
	return nil
}

func parseFlags() (*commandFlags, error) {
	flags := &commandFlags{}

	flag.StringVar(&flags.mode, "mode", "", "Operation mode: icon-extract, icon-change, icon-replace, res-add, res-extract, res-replace")
	flag.BoolVar(&flags.version, "version", false, "Display version information")
	flag.BoolVar(&flags.version, "v", false, "Display version information (short)")

	flag.StringVar(&flags.inputFile, "input", "", "Input file path")
	flag.StringVar(&flags.outputFile, "output", "", "Output file path")
	flag.IntVar(&flags.width, "width", 0, "Icon width")
	flag.IntVar(&flags.height, "height", 0, "Icon height")
	flag.IntVar(&flags.iconIndex, "index", 0, "Icon index")
	flag.BoolVar(&flags.useIndex, "use-index", false, "Use icon index")

	flag.StringVar(&flags.iconFile, "icon", "", "ICO file path")
	flag.StringVar(&flags.exeFile, "exe", "", "Executable file path")

	resType := flag.String("type", "", "Resource type (1-24)")
	resID := flag.String("id", "", "Resource ID")
	langID := flag.String("lang", "0", "Language ID (default: 0 neutral)")

	flag.Parse()

	if *resType != "" {
		val, err := strconv.ParseUint(*resType, 0, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid resource type: %v", err)
		}
		flags.resourceType = uint16(val)
	}

	if *resID != "" {
		val, err := strconv.ParseUint(*resID, 0, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid resource ID: %v", err)
		}
		flags.resourceID = uint16(val)
	}

	if *langID != "" {
		val, err := strconv.ParseUint(*langID, 0, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid language ID: %v", err)
		}
		flags.languageID = uint16(val)
	}

	return flags, nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  Icon Extract: program -mode icon-extract -input <input_file> -output <output_ico> [-width width] [-height height] [-index icon_index]")
	fmt.Println("  Icon Change:  program -mode icon-change -icon <ico_file> -exe <exe_file>")
	fmt.Println("  Icon Replace: program -mode icon-replace -input <input_exe> -output <output_exe>")
	fmt.Println("  Resource Add Res: program -mode res-add -input <exe_file> -output <res_file> -type <res_type> -id <res_id> -lang <res_lang>")
	fmt.Println("  Resource Extract: program -mode res-extract -input <exe_file> -output <res_file> -type <res_type> -id <res_id> -lang <res_lang>")
	fmt.Println("  Resource Replace: program -mode res-replace -input <input_exe> -output <output_exe> -type <res_type> -id <res_id> -lang <res_lang>")
	fmt.Println("  Demo 1: program -mode icon-replace -input calc.exe -output target.exe")
	fmt.Println("  Demo 2: program -mode res-replace  -input calc.exe -output target.exe -type 16 -id 1 -lang 0")
	fmt.Println("  Version: program -version|-v")
}

func main() {
	flags, err := parseFlags()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}

	if flags.version {
		fmt.Printf("ThiefBypassGo Version 1.0.0\n")
		os.Exit(0)
	}

	switch flags.mode {
	case "icon-extract":
		if flags.inputFile == "" || flags.outputFile == "" {
			fmt.Println("Usage: program -mode icon-extract -input <input_file> -output <output_ico> [-width width] [-height height] [-index icon_index]")
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
			fmt.Println("Usage: program -mode icon-change -icon <ico_file> -exe <exe_file>")
			os.Exit(1)
		}
		if changeExecutableIcon(flags.iconFile, flags.exeFile) {
			fmt.Println("Change icon success!")
			os.Exit(0)
		}
		fmt.Println("Change icon failed!")
		os.Exit(1)

	case "icon-replace":
		if flags.inputFile == "" || flags.outputFile == "" {
			fmt.Println("Usage: program -mode icon-replace -input <input_exe> -output <output_exe>")
			os.Exit(1)
		}
		err = ReplaceIcon(flags.inputFile, flags.outputFile)
		if err != nil {
			fmt.Printf("Failed to replace icon: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Icon replaced successfully")

	case "res-add":
		if flags.inputFile == "" || flags.outputFile == "" {
			fmt.Println("Usage: program -mode res-add -input <exe_file> -output <res_file> -type <res_type> -id <res_id> -lang <res_lang>")
			os.Exit(1)
		}
		err = AddResourceFromFile(flags.inputFile, flags.outputFile, flags.resourceType, flags.resourceID, flags.languageID)
		if err != nil {
			fmt.Printf("Failed to add resource: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Resource added successfully")

	case "res-extract":
		if flags.inputFile == "" || flags.outputFile == "" {
			fmt.Println("Usage: program -mode res-extract -input <exe_file> -output <res_file> -type <res_type> -id <res_id> -lang <res_lang>")
			os.Exit(1)
		}
		err = ExtractResourceToFile(flags.inputFile, flags.outputFile, flags.resourceType, flags.resourceID, flags.languageID)
		if err != nil {
			fmt.Printf("Failed to extract resource: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Resource extracted successfully")

	case "res-replace":
		if flags.inputFile == "" || flags.outputFile == "" {
			fmt.Println("Usage: program -mode res-replace -input <input_exe> -output <output_exe> -type <res_type> -id <res_id> -lang <res_lang>")
			os.Exit(1)
		}
		err = ReplaceResource(flags.inputFile, flags.outputFile, flags.resourceType, flags.resourceID, flags.languageID)
		if err != nil {
			fmt.Printf("Failed to replace resource: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Resource replaced successfully")

	default:
		printUsage()
		os.Exit(1)
	}
}

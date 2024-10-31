package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"unsafe"
)

/*

// 常用资源类型定义
const (
	RT_CURSOR       = 1
	RT_BITMAP       = 2
	RT_ICON         = 3
	RT_MENU         = 4
	RT_DIALOG       = 5
	RT_STRING       = 6
	RT_FONTDIR      = 7
	RT_FONT         = 8
	RT_ACCELERATOR  = 9
	RT_RCDATA       = 10
	RT_MESSAGETABLE = 11
	RT_GROUP_CURSOR = 12
	RT_GROUP_ICON   = 14
	RT_VERSION      = 16
	RT_DLGINCLUDE   = 17
	RT_PLUGPLAY     = 19
	RT_VXD          = 20
	RT_ANICURSOR    = 21
	RT_ANIICON      = 22
	RT_HTML         = 23
	RT_MANIFEST     = 24
)

// 常用语言ID定义
const (
	LANG_NEUTRAL    = 0x0000 // 中立
	LANG_ENGLISH_US = 0x0409 // 英语(美国) = 1033
	LANG_CHINESE_CN = 0x0804 // 中文(简体) = 2052
	LANG_JAPANESE   = 0x0411 // 日语 = 1041
)
*/

type CommandLine struct {
	mode         string
	input        string
	output       string
	resourceType uint16
	resourceID   uint16
	languageID   uint16
}

func parseFlags() (*CommandLine, error) {
	cmd := &CommandLine{}

	// 定义命令行参数
	flag.StringVar(&cmd.mode, "mode", "", "Operation mode: add or extract")
	flag.StringVar(&cmd.input, "input", "", "Input PE file path")
	flag.StringVar(&cmd.output, "output", "", "Output file path or resource file path")

	// 使用字符串来接收数字参数，以便后续转换
	resType := flag.String("type", "", "Resource type (1-24)")
	resID := flag.String("id", "", "Resource ID")
	langID := flag.String("lang", "0", "Language ID (default: 0 neutral)")

	// 解析命令行参数
	flag.Parse()

	// 验证必需参数
	if cmd.mode == "" || cmd.input == "" || cmd.output == "" || *resType == "" || *resID == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	// 验证模式
	if cmd.mode != "add" && cmd.mode != "extract" {
		return nil, fmt.Errorf("invalid mode: must be 'add' or 'extract'")
	}

	// 转换数字参数
	if val, err := strconv.ParseUint(*resType, 0, 16); err != nil {
		return nil, fmt.Errorf("invalid resource type: %v", err)
	} else {
		cmd.resourceType = uint16(val)
	}

	if val, err := strconv.ParseUint(*resID, 0, 16); err != nil {
		return nil, fmt.Errorf("invalid resource ID: %v", err)
	} else {
		cmd.resourceID = uint16(val)
	}

	if val, err := strconv.ParseUint(*langID, 0, 16); err != nil {
		return nil, fmt.Errorf("invalid language ID: %v", err)
	} else {
		cmd.languageID = uint16(val)
	}

	return cmd, nil
}

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procBeginUpdateResource = kernel32.NewProc("BeginUpdateResourceW")
	procUpdateResource      = kernel32.NewProc("UpdateResourceW")
	procEndUpdateResource   = kernel32.NewProc("EndUpdateResourceW")
	procLoadLibraryEx       = kernel32.NewProc("LoadLibraryExW")
	procFindResourceEx      = kernel32.NewProc("FindResourceExW")
	procLoadResource        = kernel32.NewProc("LoadResource")
	procLockResource        = kernel32.NewProc("LockResource")
	procSizeofResource      = kernel32.NewProc("SizeofResource")
	procFreeLibrary         = kernel32.NewProc("FreeLibrary")
)

const (
	LOAD_LIBRARY_AS_DATAFILE = 0x2
	LANG_NEUTRAL             = 0x0
)

// AddResourceFromFile 从文件添加资源到PE文件
func AddResourceFromFile(peFile string, resFile string, resType uint16, resName uint16, resLang uint16) error {
	// 读取资源文件
	data, err := ioutil.ReadFile(resFile)
	if err != nil {
		return fmt.Errorf("read resource file failed: %v", err)
	}

	// 转换文件名为UTF16
	peFileW, err := syscall.UTF16PtrFromString(peFile)
	if err != nil {
		return fmt.Errorf("convert filename failed: %v", err)
	}

	// 开始更新资源
	handle, _, err := procBeginUpdateResource.Call(
		uintptr(unsafe.Pointer(peFileW)),
		uintptr(0)) // false = 不删除已存在的资源

	if handle == 0 {
		return fmt.Errorf("BeginUpdateResource failed: %v", err)
	}

	// 更新资源
	success, _, err := procUpdateResource.Call(
		handle,
		uintptr(resType),
		uintptr(resName),
		uintptr(resLang),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)))

	if success == 0 {
		return fmt.Errorf("UpdateResource failed: %v", err)
	}

	// 完成更新
	success, _, err = procEndUpdateResource.Call(
		handle,
		uintptr(0)) // false = 保存更改

	if success == 0 {
		return fmt.Errorf("EndUpdateResource failed: %v", err)
	}

	return nil
}

// ExtractResourceToFile 从PE文件中提取资源到文件
func ExtractResourceToFile(peFile string, outFile string, resType uint16, resName uint16, resLang uint16) error {
	// 转换文件名为UTF16
	peFileW, err := syscall.UTF16PtrFromString(peFile)
	if err != nil {
		return fmt.Errorf("convert filename failed: %v", err)
	}

	// 加载PE文件
	hModule, _, err := procLoadLibraryEx.Call(
		uintptr(unsafe.Pointer(peFileW)),
		0,
		LOAD_LIBRARY_AS_DATAFILE)

	if hModule == 0 {
		return fmt.Errorf("LoadLibraryEx failed: %v", err)
	}
	defer procFreeLibrary.Call(hModule)

	// 查找资源
	hResInfo, _, err := procFindResourceEx.Call(
		hModule,
		uintptr(resType),
		uintptr(resName),
		uintptr(resLang))

	if hResInfo == 0 {
		return fmt.Errorf("FindResourceEx failed: %v", err)
	}

	// 加载资源
	hResData, _, err := procLoadResource.Call(hModule, hResInfo)
	if hResData == 0 {
		return fmt.Errorf("LoadResource failed: %v", err)
	}

	// 锁定资源
	lpResData, _, err := procLockResource.Call(hResData)
	if lpResData == 0 {
		return fmt.Errorf("LockResource failed: %v", err)
	}

	// 获取资源大小
	size, _, err := procSizeofResource.Call(hModule, hResInfo)
	if size == 0 {
		return fmt.Errorf("SizeofResource failed: %v", err)
	}

	// 复制资源数据
	data := make([]byte, size)
	copy(data, unsafe.Slice((*byte)(unsafe.Pointer(lpResData)), size))

	// 创建输出目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return fmt.Errorf("create output directory failed: %v", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(outFile, data, 0644); err != nil {
		return fmt.Errorf("write output file failed: %v", err)
	}

	return nil
}

func printUsage() {
	fmt.Println("\nExample:")
	fmt.Println("  program -mode add 		-input app.exe -output icon.ico -type 3 -id 1 -lang 0")
	fmt.Println("  program -mode extract 	-input app.exe -output icon.ico -type 3 -id 1 -lang 0")
}

func main() {
	// 如果没有参数，显示帮助信息
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	// 解析命令行参数
	cmd, err := parseFlags()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printUsage()
		return
	}

	// 执行相应的操作
	switch cmd.mode {
	case "add":
		err = AddResourceFromFile(cmd.input, cmd.output, cmd.resourceType, cmd.resourceID, cmd.languageID)
		if err != nil {
			fmt.Printf("Failed to add resource: %v\n", err)
			return
		}
		fmt.Println("Resource added successfully")

	case "extract":
		err = ExtractResourceToFile(cmd.input, cmd.output, cmd.resourceType, cmd.resourceID, cmd.languageID)
		if err != nil {
			fmt.Printf("Failed to extract resource: %v\n", err)
			return
		}
		fmt.Println("Resource extracted successfully")
	}
}

package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"unicode/utf16"
)

type VSFixedFileInfo struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}

type FileInfo struct {
	ProductName      string
	FileDescription  string
	CompanyName      string
	LegalCopyright   string
	OriginalFilename string
	ProductVersion   string
	FileVersion      string
	ModifyTime       time.Time
}

func readUTF16String(data []byte) string {
	if len(data) < 2 {
		return ""
	}

	// 查找null终止符
	var end int
	for end = 0; end < len(data)-1; end += 2 {
		if data[end] == 0 && data[end+1] == 0 {
			break
		}
	}

	u16str := make([]uint16, len(data[:end])/2)
	for i := range u16str {
		u16str[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(u16str))
}

func findStringValue(block []byte, key string) string {
	keyUTF16 := utf16.Encode([]rune(key))
	keyBytes := make([]byte, len(keyUTF16)*2)
	for i, v := range keyUTF16 {
		binary.LittleEndian.PutUint16(keyBytes[i*2:], v)
	}

	// 在块中查找键
	for i := 0; i < len(block)-len(keyBytes); i++ {
		if bytes.Equal(block[i:i+len(keyBytes)], keyBytes) {
			// 找到键后，跳过键和填充字节，读取值
			valueStart := i + len(keyBytes)
			for valueStart < len(block) && block[valueStart] == 0 {
				valueStart++
			}
			if valueStart < len(block) {
				return readUTF16String(block[valueStart:])
			}
		}
	}
	return ""
}

func getFileDetails(filename string) (*FileInfo, error) {
	file, err := pe.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// 获取文件修改时间
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	info := &FileInfo{
		ModifyTime: fileInfo.ModTime(),
	}

	// 查找资源节
	var rsrcSection *pe.Section
	for _, section := range file.Sections {
		if section.Name == ".rsrc" {
			rsrcSection = section
			break
		}
	}

	if rsrcSection == nil {
		return info, nil
	}

	data, err := rsrcSection.Data()
	if err != nil {
		return info, fmt.Errorf("failed to read resource section: %v", err)
	}

	// 查找VERSION_INFO资源
	var versionInfo []byte
	for i := 0; i < len(data)-8; i++ {
		if binary.LittleEndian.Uint32(data[i:]) == 0xFEEF04BD {
			versionInfo = data[i:]
			break
		}
	}

	if len(versionInfo) > 0 {
		var fixedInfo VSFixedFileInfo
		reader := bytes.NewReader(versionInfo)
		if err := binary.Read(reader, binary.LittleEndian, &fixedInfo); err == nil {
			// 设置文件版本
			info.FileVersion = fmt.Sprintf("%d.%d.%d.%d",
				fixedInfo.FileVersionMS>>16,
				fixedInfo.FileVersionMS&0xFFFF,
				fixedInfo.FileVersionLS>>16,
				fixedInfo.FileVersionLS&0xFFFF)

			// 设置产品版本
			info.ProductVersion = fmt.Sprintf("%d.%d.%d.%d",
				fixedInfo.ProductVersionMS>>16,
				fixedInfo.ProductVersionMS&0xFFFF,
				fixedInfo.ProductVersionLS>>16,
				fixedInfo.ProductVersionLS&0xFFFF)
		}

		// 提取字符串信息
		info.ProductName = findStringValue(versionInfo, "ProductName")
		info.FileDescription = findStringValue(versionInfo, "FileDescription")
		info.CompanyName = findStringValue(versionInfo, "CompanyName")
		info.LegalCopyright = findStringValue(versionInfo, "LegalCopyright")
		info.OriginalFilename = findStringValue(versionInfo, "OriginalFilename")
	}

	return info, nil
}

func copyFileDetails(sourcePath, targetPath string) error {
	// 首先读取源文件的版本信息
	sourceInfo, err := getFileDetails(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file details: %v", err)
	}

	// 打开目标文件
	targetFile, err := os.OpenFile(targetPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("failed to open target file: %v", err)
	}
	defer targetFile.Close()

	// 读取目标文件的PE信息
	peFile, err := pe.NewFile(targetFile)
	if err != nil {
		return fmt.Errorf("failed to parse target PE file: %v", err)
	}

	// 查找资源节
	var rsrcSection *pe.Section
	for _, section := range peFile.Sections {
		if section.Name == ".rsrc" {
			rsrcSection = section
			break
		}
	}

	if rsrcSection == nil {
		return fmt.Errorf("no resource section found in target file")
	}

	// 读取完整的资源节数据
	data, err := rsrcSection.Data()
	if err != nil {
		return fmt.Errorf("failed to read resource section: %v", err)
	}

	// 查找VERSION_INFO资源的位置
	var versionStart int = -1
	for i := 0; i < len(data)-8; i++ {
		if binary.LittleEndian.Uint32(data[i:]) == 0xFEEF04BD {
			versionStart = i
			break
		}
	}

	if versionStart == -1 {
		return fmt.Errorf("version information not found in target file")
	}

	// 创建新的版本信息块
	var newVersionInfo bytes.Buffer

	// 写入VS_FIXEDFILEINFO结构
	fileVersionMS := uint32(0)
	fileVersionLS := uint32(0)
	productVersionMS := uint32(0)
	productVersionLS := uint32(0)

	// 解析版本号字符串
	fmt.Sscanf(sourceInfo.FileVersion, "%d.%d.%d.%d",
		&fileVersionMS, &fileVersionLS, &productVersionMS, &productVersionLS)

	fixedInfo := VSFixedFileInfo{
		Signature:        0xFEEF04BD,
		StrucVersion:     0x00010000,
		FileVersionMS:    (fileVersionMS << 16) | fileVersionLS,
		FileVersionLS:    (productVersionMS << 16) | productVersionLS,
		ProductVersionMS: (fileVersionMS << 16) | fileVersionLS,
		ProductVersionLS: (productVersionMS << 16) | productVersionLS,
		FileFlagsMask:    0x3F,
		FileFlags:        0,
		FileOS:           0x40004,
		FileType:         1,
		FileSubtype:      0,
		FileDateMS:       0,
		FileDateLS:       0,
	}

	// 写入固定版本信息
	binary.Write(&newVersionInfo, binary.LittleEndian, fixedInfo)

	// 创建字符串信息表
	stringPairs := []struct {
		key   string
		value string
	}{
		{"ProductName", sourceInfo.ProductName},
		{"FileDescription", sourceInfo.FileDescription},
		{"CompanyName", sourceInfo.CompanyName},
		{"LegalCopyright", sourceInfo.LegalCopyright},
		{"OriginalFilename", sourceInfo.OriginalFilename},
		{"ProductVersion", sourceInfo.ProductVersion},
		{"FileVersion", sourceInfo.FileVersion},
	}

	// 为每个字符串对创建UTF-16编码
	for _, pair := range stringPairs {
		keyRunes := utf16.Encode([]rune(pair.key))
		valueRunes := utf16.Encode([]rune(pair.value))

		// 写入键
		for _, r := range keyRunes {
			binary.Write(&newVersionInfo, binary.LittleEndian, r)
		}
		// 写入null终止符
		binary.Write(&newVersionInfo, binary.LittleEndian, uint16(0))

		// 写入值
		for _, r := range valueRunes {
			binary.Write(&newVersionInfo, binary.LittleEndian, r)
		}
		// 写入null终止符
		binary.Write(&newVersionInfo, binary.LittleEndian, uint16(0))

		// 4字节对齐
		padding := make([]byte, (4-(newVersionInfo.Len()%4))%4)
		newVersionInfo.Write(padding)
	}

	// 定位到资源节在文件中的位置
	_, err = targetFile.Seek(int64(rsrcSection.Offset+uint32(versionStart)), io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to version info: %v", err)
	}

	// 写入新的版本信息
	_, err = targetFile.Write(newVersionInfo.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write new version info: %v", err)
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "-info":
		if len(os.Args) != 3 {
			fmt.Println("错误: -info 命令需要一个文件路径参数")
			printUsage()
			os.Exit(1)
		}
		// 显示文件信息
		info, err := getFileDetails(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("产品名称: %s\n", info.ProductName)
		fmt.Printf("文件说明: %s\n", info.FileDescription)
		fmt.Printf("公司名称: %s\n", info.CompanyName)
		fmt.Printf("版权信息: %s\n", info.LegalCopyright)
		fmt.Printf("原始文件名: %s\n", info.OriginalFilename)
		fmt.Printf("产品版本: %s\n", info.ProductVersion)
		fmt.Printf("文件版本: %s\n", info.FileVersion)
		fmt.Printf("修改时间: %s\n", info.ModifyTime.Format("2006-01-02 15:04:05"))

	case "-copy":
		if len(os.Args) != 4 {
			fmt.Println("错误: -copy 命令需要源文件和目标文件两个参数")
			printUsage()
			os.Exit(1)
		}
		// 复制文件信息
		err := copyFileDetails(os.Args[2], os.Args[3])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("文件信息复制成功！")

	default:
		fmt.Printf("错误: 未知的命令 '%s'\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  查看文件信息: program -info <filename>")
	fmt.Println("  复制文件信息: program -copy <source_file> <target_file>")
}

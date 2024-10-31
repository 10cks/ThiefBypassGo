package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
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

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  查看文件信息: program -info <filename>")
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

	default:
		fmt.Printf("错误: 未知的命令 '%s'\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

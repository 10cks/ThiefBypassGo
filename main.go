package main

import (
	"fmt"
	"github.com/orcastor/fico"
	"os"
)

// ConvertToICO 将输入文件转换为ICO格式
// inputPath: 输入文件路径
// outputPath: 输出文件路径
// width: 图标宽度(0表示包含所有尺寸)
// height: 图标高度(0表示包含所有尺寸)
// iconIndex: 图标索引(仅用于PE文件，nil表示包含所有图标)
func ConvertToICO(inputPath, outputPath string, width, height int, iconIndex *int) error {
	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 配置选项
	cfg := fico.Config{
		Format: "ico",     // 输出格式为ico
		Width:  width,     // 指定宽度
		Height: height,    // 指定高度
		Index:  iconIndex, // 图标索引
	}

	// 将输入文件转换为ICO
	return fico.F2ICO(outFile, inputPath, cfg)
}

func main() {
	// 转换EXE文件，提取32x32尺寸的第一个图标
	index := 0
	err := ConvertToICO("calc.exe", "calc.ico", 32, 32, &index)
	if err != nil {
		panic(err)
	} else {
		fmt.Println("Exact Success!")
	}
}

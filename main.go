package main

import (
	"github.com/orcastor/fico"
	"os"
)

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

//func main() {
//	// 转换EXE文件，提取32x32尺寸的第一个图标
//	index := 0
//	err := ConvertToICO("calc.exe", "calc.ico", 32, 32, &index)
//	if err != nil {
//		panic(err)
//	} else {
//		fmt.Println("Exact Success!")
//	}
//}

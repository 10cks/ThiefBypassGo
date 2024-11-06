# ThiefBypassGo

主程序为`cmd/main.go`，该程序用于迅速拷贝程序信息到指定程序中，也可用于提取相关资源。

### 图标相关功能

- **提取图标**: 从输入文件中提取图标并保存为 ICO 文件。
  ```
  用法: program -mode icon-extract -input <输入文件> -output <输出ICO文件> [-width 宽度] [-height 高度] [-index 图标索引]
  ```

- **更改图标**: 将 ICO 文件中的图标应用到可执行文件。
  ```
  用法: program -mode icon-change -icon <ICO文件> -exe <可执行文件>
  ```

- **替换图标**: 从一个可执行文件中提取图标并替换到另一个可执行文件中。
  ```
  用法: program -mode icon-replace -input <输入可执行文件> -output <输出可执行文件>
  ```

### 资源相关功能

- **添加资源**: 将资源文件添加到可执行文件中。
  ```
  用法: program -mode res-add -input <可执行文件> -output <资源文件> -type <资源类型> -id <资源ID> -lang <语言ID>
  ```

- **提取资源**: 从可执行文件中提取资源并保存到文件。
  ```
  用法: program -mode res-extract -input <可执行文件> -output <资源文件> -type <资源类型> -id <资源ID> -lang <语言ID>
  ```

- **替换资源**: 从一个可执行文件中提取资源并将其添加到另一个可执行文件中。
  ```
  用法: program -mode res-replace -input <输入可执行文件> -output <输出可执行文件> -type <资源类型> -id <资源ID> -lang <语言ID>
  ```

### 示例用法

- **示例 1**: 用 `calc.exe` 的图标替换 `target.exe` 的图标。
  ```
  用法: program -mode icon-replace -input calc.exe -output target.exe
  ```

- **示例 2**: 用 `calc.exe` 的指定资源替换 `target.exe` 的资源。
  ```
  用法: program -mode res-replace -input calc.exe -output target.exe -type 16 -id 1 -lang 0
  ```

### 版本信息

- **查看版本**:
  ```
  用法: program -version|-v
  ```
### Usage

```powershell
Usage:
  Icon Extract: program -mode icon-extract -input <input_file> -output <output_ico> [-width width] [-height height] [-index icon_index]
  Icon Change:  program -mode icon-change -icon <ico_file> -exe <exe_file>
  Icon Replace: program -mode icon-replace -input <input_exe> -output <output_exe>
  Resource Add Res: program -mode res-add -input <exe_file> -output <res_file> -type <res_type> -id <res_id> -lang <res_lang>
  Resource Extract: program -mode res-extract -input <exe_file> -output <res_file> -type <res_type> -id <res_id> -lang <res_lang>
  Resource Replace: program -mode res-replace -input <input_exe> -output <output_exe> -type <res_type> -id <res_id> -lang <res_lang>
  Demo 1: program -mode icon-replace -input calc.exe -output target.exe
  Demo 2: program -mode res-replace  -input calc.exe -output target.exe -type 16 -id 1 -lang 0
  Version: program -version|-v
```

资源文件替换列表（勾选为常用参数）：
```powershell
// 常用资源类型定义（type）
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
	RT_VERSION      = 16 √
	RT_DLGINCLUDE   = 17
	RT_PLUGPLAY     = 19
	RT_VXD          = 20
	RT_ANICURSOR    = 21
	RT_ANIICON      = 22
	RT_HTML         = 23
	RT_MANIFEST     = 24 √
)

// 常用语言（id）
const (
	LANG_NEUTRAL    = 0
	LANG_ENGLISH_US = 1033 // 英语(美国)
	LANG_CHINESE_CN = 2052 // 中文(简体)
	LANG_JAPANESE   = 1041 // 日语
)
```
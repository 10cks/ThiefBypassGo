```powershell
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
	RT_VERSION      = 16 √
	RT_DLGINCLUDE   = 17
	RT_PLUGPLAY     = 19
	RT_VXD          = 20
	RT_ANICURSOR    = 21
	RT_ANIICON      = 22
	RT_HTML         = 23
	RT_MANIFEST     = 24 √
)

// 常用语言ID定义
const (
	LANG_NEUTRAL    = 0x0000 // 中立
	LANG_ENGLISH_US = 0x0409 // 英语(美国) = 1033
	LANG_CHINESE_CN = 0x0804 // 中文(简体) = 2052
	LANG_JAPANESE   = 0x0411 // 日语 = 1041
)
```
# ThiefBypassGo

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
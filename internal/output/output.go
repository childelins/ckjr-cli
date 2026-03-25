package output

import (
	"encoding/json"
	"io"
)

// Printer 控制输出格式
type Printer struct {
	Pretty bool
	Writer io.Writer
}

// Print 输出 JSON 数据
func Print(w io.Writer, data interface{}, pretty bool) {
	var bytes []byte
	var err error
	if pretty {
		bytes, err = json.MarshalIndent(data, "", "  ")
	} else {
		bytes, err = json.Marshal(data)
	}
	if err != nil {
		PrintError(w, err.Error())
		return
	}
	w.Write(bytes)
	w.Write([]byte("\n"))
}

// PrintError 输出错误信息
func PrintError(w io.Writer, msg string) {
	data, _ := json.Marshal(map[string]string{"error": msg})
	w.Write(data)
	w.Write([]byte("\n"))
}

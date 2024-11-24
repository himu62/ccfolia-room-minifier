package main

import (
	"syscall/js"

	"github.com/himu62/ccfolia-room-minifier/process"
)

func main() {
	js.Global().Set("processZipWasm", js.FuncOf(processZipWasm))
	select {}
}

func processZipWasm(this js.Value, args []js.Value) interface{} {
	return js.Global().Get("Promise").New(js.FuncOf(func(resolve, reject js.Value) interface{} {
		go func() {
			if len(args) < 1 {
				reject.Invoke("ファイルが選択されていません")
				return
			}

			input := args[0]
			data := make([]byte, input.Get("length").Int())
			js.CopyBytesToGo(data, input)

			outputData, err := process.ProcessZip(data)
			if err != nil {
				reject.Invoke(err.Error())
				return
			}

			result := js.Global().Get("Uint8Array").New(len(outputData))
			js.CopyBytesToJS(result, outputData)
			resolve.Invoke(result)
		}()
		return nil
	}))
}

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/himu62/ccfolia-room-minifier/compare"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: compare <filename>")
		os.Exit(1)
	}
	inputPath := os.Args[1]

	params := map[string]compare.Param{
		"q70-2048": {WebPQuality: 70, Quantize: true, ColorPalette: 2048},
		"q70":      {WebPQuality: 70, Quantize: false},
		"q70-1024": {WebPQuality: 70, Quantize: true, ColorPalette: 1024},
		"q80":      {WebPQuality: 80, Quantize: false},
	}

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	for name, param := range params {
		now := time.Now()
		outputPath := inputPath + "_" + name + ".webp"

		outputData, err := compare.ProcessImage(inputData, &param)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		err = os.WriteFile(outputPath, outputData, 0644)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("Done %s: %s (size: %s)\n", name, time.Since(now), humanizeSize(len(outputData)))
	}
}

func humanizeSize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

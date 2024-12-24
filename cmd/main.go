package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/himu62/ccfolia-room-minifier/process"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ccfolia-room-minifier <filename>")
		os.Exit(1)
	}
	inputPath := os.Args[1]
	outputPath := generateOutputPath(inputPath)

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	outputData, err := process.ProcessZip(inputData)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	err = os.WriteFile(outputPath, outputData, 0644)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("\nDone")
}

func generateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	outputName := name + "_compressed" + ext
	return filepath.Join(dir, outputName)
}

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	webp "github.com/kolesa-team/go-webp/encoder"
)

const (
	imageExts = ".png .jpg .jpeg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ccfolia-room-minifier <filename>")
		os.Exit(1)
	}
	inputPath := os.Args[1]
	outputPath := generateOutputPath(inputPath)

	err := processZip(inputPath, outputPath)
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

func processZip(inputPath, outputPath string) error {
	reader, err := zip.OpenReader(inputPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	fileData := make(map[string][]byte)
	filenameMap := make(map[string]string)

	var readSize int
	var reducedSize int
	for i, file := range reader.File {
		data, err := readFile(file)
		if err != nil {
			return err
		}

		if !isImageFile(file.Name) {
			fileData[file.Name] = data
			continue
		}

		animated, err := isAnimatedImage(data, file.Name)
		if err != nil {
			return err
		}
		if animated {
			fileData[file.Name] = data
			continue
		}

		newData, err := processImage(data)
		if err != nil {
			return err
		}

		hash := sha256.Sum256(newData)
		newFilename := fmt.Sprintf("%x.webp", hash)
		fileData[newFilename] = newData
		filenameMap[file.Name] = newFilename

		readSize += len(data)
		reducedSize += len(data) - len(newData)
		fmt.Printf("\rProcessing... %d/%d (reduced %s, %.0f%%) ", i+1, len(reader.File), humanizeSize(reducedSize), 100-float64(reducedSize)/float64(readSize)*100)
	}

	if data, exists := fileData["__data.json"]; exists {
		newData, err := processDataJSON(data, filenameMap)
		if err != nil {
			return err
		}
		fileData["__data.json"] = newData

		hash := sha256.Sum256(newData)
		fileData[".token"] = []byte(fmt.Sprintf("0.%x", hash))
	} else {
		return fmt.Errorf("__data.json not found")
	}

	return writeZip(outputPath, fileData)
}

func readFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	err = rc.Close()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func isImageFile(filename string) bool {
	ext := filepath.Ext(filename)
	for _, imageExt := range strings.Split(imageExts, " ") {
		if ext == imageExt {
			return true
		}
	}
	return false
}

func isAnimatedImage(data []byte, filename string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".png" {
		return isAnimatedPNG(data)
	}
	return false, nil
}

func isAnimatedPNG(data []byte) (bool, error) {
	const (
		pngSignature = "\x89PNG\r\n\x1a\n"
	)

	reader := bytes.NewReader(data)

	sig := make([]byte, 8)
	_, err := io.ReadFull(reader, sig)
	if err != nil {
		return false, err
	}
	if string(sig) != pngSignature {
		return false, fmt.Errorf("invalid PNG image")
	}

	for {
		var chunkLen int32
		err := binary.Read(reader, binary.BigEndian, &chunkLen)
		if err != nil {
			break
		}

		chunkType := make([]byte, 4)
		_, err = io.ReadFull(reader, chunkType)
		if err != nil {
			return false, err
		}

		if string(chunkType) == "acTL" {
			return true, nil
		}

		_, err = reader.Seek(int64(chunkLen)+4, io.SeekCurrent)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func processImage(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	options, err := webp.NewLossyEncoderOptions(webp.PresetPicture, 75)
	if err != nil {
		return nil, err
	}
	options.Method = 6

	encoder, err := webp.NewEncoder(img, options)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = encoder.Encode(&buf)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("image processed (%s -> %s, %.0f%%)\n", humanizeSize(len(data)), humanizeSize(buf.Len()), float64(buf.Len())/float64(len(data))*100)
	return buf.Bytes(), nil
}

func humanizeSize(size int) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case size < kb:
		return fmt.Sprintf("%dB", size)
	case size < mb:
		return fmt.Sprintf("%.0fKB", float64(size)/kb)
	case size < gb:
		return fmt.Sprintf("%.0fMB", float64(size)/mb)
	default:
		return fmt.Sprintf("%.0fGB", float64(size)/gb)
	}
}

func processDataJSON(data []byte, filenameMap map[string]string) ([]byte, error) {
	jsonStr := string(data)
	for oldFilename, newFilename := range filenameMap {
		jsonStr = strings.ReplaceAll(jsonStr, oldFilename, newFilename)
	}

	var newJSON map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &newJSON)
	if err != nil {
		return nil, err
	}

	if resources, exists := newJSON["resources"].(map[string]interface{}); exists {
		newResources := make(map[string]interface{})
		for filename, resource := range resources {
			if checkFileProcessed(filename, filenameMap) {
				newResources[filename] = map[string]interface{}{
					"type": "image/webp",
				}
			} else {
				newResources[filename] = resource
			}
		}
		newJSON["resources"] = newResources
	}

	newData, err := json.Marshal(newJSON)
	if err != nil {
		return nil, err
	}
	return newData, nil
}

func checkFileProcessed(newFilename string, filenameMap map[string]string) bool {
	for _, name := range filenameMap {
		if newFilename == name {
			return true
		}
	}
	return false
}

func writeZip(path string, fileData map[string][]byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	var i int
	for filename, data := range fileData {
		i++
		fmt.Printf("\rWriting... %d/%d ", i+1, len(fileData))

		writer, err := writer.Create(filename)
		if err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
	}

	return nil
}

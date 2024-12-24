package process

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color/palette"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/esimov/colorquant"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

const (
	imageExts    = ".png .jpg .jpeg"
	colorPalette = 2048
	webpQuality  = 70
)

var ditherer = colorquant.Dither{
	Filter: [][]float32{
		{0.0, 0.0, 0.0, 7.0 / 48.0, 5.0 / 48.0},
		{3.0 / 48.0, 5.0 / 48.0, 7.0 / 48.0, 5.0 / 48.0, 3.0 / 48.0},
		{1.0 / 48.0, 3.0 / 48.0, 5.0 / 48.0, 3.0 / 48.0, 1.0 / 48.0},
	},
}

func ProcessZip(inputData []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(inputData), int64(len(inputData)))
	if err != nil {
		return nil, err
	}

	fileData := make(map[string][]byte)
	filenameMap := make(map[string]string)

	var readSize int
	var reducedSize int
	for i, file := range reader.File {
		data, err := readFile(file)
		if err != nil {
			return nil, err
		}

		if !isImageFile(file.Name) {
			fileData[file.Name] = data
			continue
		}

		animated, err := isAnimatedImage(data, file.Name)
		if err != nil {
			return nil, err
		}
		if animated {
			fileData[file.Name] = data
			continue
		}

		newData, err := processImage(data)
		if err != nil {
			return nil, err
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
			return nil, err
		}
		fileData["__data.json"] = newData

		hash := sha256.Sum256(newData)
		fileData[".token"] = []byte(fmt.Sprintf("0.%x", hash))
	} else {
		return nil, fmt.Errorf("__data.json not found")
	}

	outputData, err := writeZip(fileData)
	if err != nil {
		return nil, err
	}
	return outputData, nil
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

	quantized := image.NewPaletted(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()), palette.WebSafe)
	quant := ditherer.Quantize(img, quantized, colorPalette, false, true)

	options, err := encoder.NewLossyEncoderOptions(encoder.PresetPicture, webpQuality)
	if err != nil {
		return nil, err
	}
	options.Method = 6

	var buf bytes.Buffer
	err = webp.Encode(&buf, quant, options)
	if err != nil {
		return nil, err
	}
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

func writeZip(fileData map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	var i int
	for filename, data := range fileData {
		i++
		fmt.Printf("\rWriting... %d/%d ", i+1, len(fileData))

		file, err := writer.Create(filename)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write(data); err != nil {
			return nil, err
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

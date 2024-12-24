package process

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"golang.org/x/sync/errgroup"
)

const (
	imageExts    = ".png .jpg .jpeg"
	colorPalette = 2048
	webpQuality  = 70
)

func ProcessZip(inputData []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(inputData), int64(len(inputData)))
	if err != nil {
		return nil, err
	}

	fileData := make(map[string][]byte)
	filenameMap := make(map[string]string)

	for _, file := range reader.File {
		data, err := readFile(file)
		if err != nil {
			return nil, err
		}
		fileData[file.Name] = data
	}

	if _, exists := fileData["__data.json"]; !exists {
		return nil, fmt.Errorf("__data.json not found")
	}
	if _, exists := fileData[".token"]; !exists {
		return nil, fmt.Errorf(".token not found")
	}

	progressCh := make(chan string)
	total := len(fileData) - 2

	go func() {
		doneCount := 0
		for range progressCh {
			doneCount++
			fmt.Printf("\rProcessing... %d/%d ", doneCount, total)
		}
	}()

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(runtime.NumCPU() / 2)

	for name, data := range fileData {
		g.Go(func() error {
			if !isImageFile(name) {
				return nil
			}
			animated, err := isAnimatedImage(data, name)
			if err != nil {
				return err
			}
			if animated {
				return nil
			}

			newData, err := processImage(data)
			if err != nil {
				return err
			}

			hash := sha256.Sum256(newData)
			newFilename := fmt.Sprintf("%x.webp", hash)
			filenameMap[name] = newFilename
			delete(fileData, name)
			fileData[newFilename] = newData

			select {
			case <-ctx.Done():
				return ctx.Err()
			case progressCh <- name:
			}

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	close(progressCh)

	newData, err := processDataJSON(fileData["__data.json"], filenameMap)
	if err != nil {
		return nil, err
	}
	fileData["__data.json"] = newData

	hash := sha256.Sum256(newData)
	fileData[".token"] = []byte(fmt.Sprintf("0.%x", hash))

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

	options, err := encoder.NewLossyEncoderOptions(encoder.PresetPicture, webpQuality)
	if err != nil {
		return nil, err
	}
	options.Method = 6

	var buf bytes.Buffer
	err = webp.Encode(&buf, img, options)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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

package compare

import (
	"bytes"
	"image"
	"image/color/palette"
	_ "image/jpeg"
	_ "image/png"

	"github.com/esimov/colorquant"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

var ditherer = colorquant.Dither{
	Filter: [][]float32{
		{0.0, 0.0, 0.0, 7.0 / 48.0, 5.0 / 48.0},
		{3.0 / 48.0, 5.0 / 48.0, 7.0 / 48.0, 5.0 / 48.0, 3.0 / 48.0},
		{1.0 / 48.0, 3.0 / 48.0, 5.0 / 48.0, 3.0 / 48.0, 1.0 / 48.0},
	},
}

type Param struct {
	WebPQuality  float32
	Quantize     bool
	ColorPalette int
}

func ProcessImage(data []byte, param *Param) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	quant := img
	if param.Quantize {
		quantized := image.NewPaletted(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()), palette.WebSafe)
		quant = ditherer.Quantize(img, quantized, param.ColorPalette, false, true)
	}

	options, err := encoder.NewLossyEncoderOptions(encoder.PresetPicture, param.WebPQuality)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = webp.Encode(&buf, quant, options)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

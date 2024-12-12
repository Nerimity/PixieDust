package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/discord/lilliput"
	"github.com/disintegration/imaging"
	"github.com/spf13/cobra"
)

type Options struct {
	CropWidth  int
	CropHeight int
	CropX      int
	CropY      int

	DoFillResize    bool
	ResizeWidth     int
	ResizeHeight    int
	GifResizeWidth  int
	GifResizeHeight int

	ImageQuality int
}

func decodeImage(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	switch strings.ToLower(filePath[strings.LastIndex(filePath, ".")+1:]) {
	case "jpg", "jpeg":
		return jpeg.Decode(file)
	case "png":
		return png.Decode(file)
	case "webp":
		return webp.Decode(file)
	default:
		return nil, fmt.Errorf("unsupported image format")
	}
}

func encodeWebP(img image.Image, filePath string) error {
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	options := &webp.Options{Lossless: false, Quality: 100}
	return webp.Encode(outFile, img, options)
}

func cropImage(inputPath, outputPath string, opts Options) error {
	src, err := decodeImage(inputPath)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	bounds := src.Bounds()

	if opts.CropX == 0 && opts.CropY == 0 {
		opts.CropX = bounds.Max.X / 2
		opts.CropY = bounds.Max.Y / 2
	}

	if opts.CropX-opts.CropWidth/2 < 0 || opts.CropY-opts.CropHeight/2 < 0 ||
		opts.CropX+opts.CropWidth/2 > bounds.Max.X || opts.CropY+opts.CropHeight/2 > bounds.Max.Y {
		return fmt.Errorf("crop dimensions and coordinates are out of image bounds")
	}

	// rect calc
	x0 := opts.CropX - opts.CropWidth/2
	y0 := opts.CropY - opts.CropHeight/2
	x1 := opts.CropX + opts.CropWidth/2
	y1 := opts.CropY + opts.CropHeight/2
	cropRect := image.Rect(x0, y0, x1, y1)

	croppedImage := imaging.Crop(src, cropRect)

	if strings.ToLower(outputPath[strings.LastIndex(outputPath, ".")+1:]) == "webp" {
		return encodeWebP(croppedImage, outputPath)
	}
	return imaging.Save(croppedImage, outputPath)
}

func compressImage(inputPath, outputPath string, opts Options) error {
	buf, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("unable to read image: %v", err)
	}

	decoder, err := lilliput.NewDecoder(buf)
	if err != nil {
		return fmt.Errorf("error decoding image: %v", err)
	}
	defer decoder.Close()

	header, err := decoder.Header()
	if err != nil {
		return fmt.Errorf("error getting image header: %v", err)
	}

	width, height := header.Width(), header.Height()
	var newWidth, newHeight int

	maxEncodeTime, err := time.ParseDuration("30s")
	if err != nil {
		return fmt.Errorf("error parsing duration: %v", err)
	}

	if header.IsAnimated() {
		newWidth, newHeight = resizeWithAspectRatio(width, height, opts.GifResizeWidth, opts.GifResizeHeight, opts)
	} else {
		newWidth, newHeight = resizeWithAspectRatio(width, height, opts.ResizeWidth, opts.ResizeHeight, opts)
	}

	ops := lilliput.NewImageOps(8192)
	defer ops.Close()

	outBuf := make([]byte, 50*1024*1024)

	resizeMethod := lilliput.ImageOpsFit

	if opts.DoFillResize {
		resizeMethod = lilliput.ImageOpsResize
	}

	imageOpts := &lilliput.ImageOptions{
		FileType:             ".webp",
		Width:                newWidth,
		Height:               newHeight,
		ResizeMethod:         resizeMethod,
		EncodeTimeout:        maxEncodeTime,
		NormalizeOrientation: true,
		EncodeOptions: map[int]int{
			lilliput.WebpQuality: opts.ImageQuality,
		},
	}

	output, err := ops.Transform(decoder, imageOpts, outBuf)
	if err != nil {
		return fmt.Errorf("error transforming image: %v", err)
	}

	err = os.WriteFile(outputPath, output, 0644)
	if err != nil {
		return fmt.Errorf("unable to write output file: %v", err)
	}

	fmt.Println("Image converted and saved to", outputPath)
	return nil
}

func resizeWithAspectRatio(origWidth, origHeight, maxWidth, maxHeight int, opts Options) (int, int) {
	if opts.DoFillResize {
		return opts.ResizeWidth, opts.ResizeHeight
	}

	if origWidth <= maxWidth && origHeight <= maxHeight {
		return origWidth, origHeight
	}

	ratio := float64(origWidth) / float64(origHeight)
	if int(float64(maxWidth)/ratio) > maxHeight {
		return int(float64(maxHeight) * ratio), maxHeight
	}
	return maxWidth, int(float64(maxWidth) / ratio)
}

func main() {
	var cropWidth, cropHeight, cropX, cropY,
		resizeWidth, resizeHeight, gifResizeWidth,
		gifResizeHeight, quality int
	var inputPath, outputPath string
	var doCrop, doFillResize bool

	rootCmd := &cobra.Command{
		Use:   "pixiedust",
		Short: "A CLI for processing images - Magically fast!",
		Run: func(cmd *cobra.Command, args []string) {
			if doCrop {
				if cropWidth <= 0 || cropHeight <= 0 || cropX < 0 || cropY < 0 {
					log.Fatal("Invalid crop parameters.")
				}

				opts := Options{
					CropWidth:  cropWidth,
					CropHeight: cropHeight,
					CropX:      cropX,
					CropY:      cropY,
				}

				err := cropImage(inputPath, outputPath, opts)
				inputPath = outputPath
				if err != nil {
					log.Fatalf("Error cropping image: %v", err)
				}
			}

			opts := Options{
				ResizeWidth:     resizeWidth,
				ResizeHeight:    resizeHeight,
				GifResizeWidth:  gifResizeWidth,
				GifResizeHeight: gifResizeHeight,
				DoFillResize:    doFillResize,
			}
			err := compressImage(inputPath, outputPath, opts)
			if err != nil {
				log.Fatalf("Error compressing image: %v", err)
			}

			fmt.Println("Image processed and saved to", outputPath)
		},
	}

	rootCmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input image path (required)")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output image path (required)")

	rootCmd.Flags().IntVar(&quality, "quality", 30, "The end quality of the WebP when re-encoding. Higher means less processing time, but also a bigger file size.")

	rootCmd.Flags().IntVar(&cropWidth, "crop-width", 0, "Crop width")
	rootCmd.Flags().IntVar(&cropHeight, "crop-height", 0, "Crop height")
	rootCmd.Flags().IntVar(&cropX, "crop-x", 0, "Crop center X coordinate")
	rootCmd.Flags().IntVar(&cropY, "crop-y", 0, "Crop center Y coordinate")
	rootCmd.Flags().BoolVar(&doCrop, "crop", false, "Crop Image")

	rootCmd.Flags().IntVar(&resizeWidth, "width", 1920, "The width to resize to")
	rootCmd.Flags().IntVar(&resizeHeight, "height", 1080, "The height to resize to")
	rootCmd.Flags().IntVar(&gifResizeWidth, "gif-width", 800, "The width to resize gifs to")
	rootCmd.Flags().IntVar(&gifResizeHeight, "gif-height", 600, "The height to resize gifs to")
	rootCmd.Flags().BoolVar(&doFillResize, "resize-fill", false, "Uses the fill method to resize the image, instead of calculating aspect ratio to make it fit.")

	rootCmd.MarkFlagRequired("input")
	rootCmd.MarkFlagRequired("output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

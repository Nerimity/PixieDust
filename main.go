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

	options := &webp.Options{Lossless: false, Quality: 90}
	return webp.Encode(outFile, img, options)
}

func processImage(inputPath, outputPath string, opts Options) error {
	src, err := decodeImage(inputPath)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Ensure the cropping coordinates and dimensions are within the image bounds
	bounds := src.Bounds()

	// If CropX and CropY are 0, crop from the center
	if opts.CropX == 0 && opts.CropY == 0 {
		opts.CropX = bounds.Max.X / 2
		opts.CropY = bounds.Max.Y / 2
	}

	if opts.CropX-opts.CropWidth/2 < 0 || opts.CropY-opts.CropHeight/2 < 0 ||
		opts.CropX+opts.CropWidth/2 > bounds.Max.X || opts.CropY+opts.CropHeight/2 > bounds.Max.Y {
		return fmt.Errorf("crop dimensions and coordinates are out of image bounds")
	}

	// Calculate the rectangle for cropping
	x0 := opts.CropX - opts.CropWidth/2
	y0 := opts.CropY - opts.CropHeight/2
	x1 := opts.CropX + opts.CropWidth/2
	y1 := opts.CropY + opts.CropHeight/2
	cropRect := image.Rect(x0, y0, x1, y1)

	// Crop the image
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
	var maxWidth, maxHeight int

	maxEncodeTime, err := time.ParseDuration("30s")
	if err != nil {
		return fmt.Errorf("error parsing duration: %v", err)
	}

	if header.IsAnimated() {
		maxWidth, maxHeight = resizeWithAspectRatio(width, height, 800, 600)
	} else {
		maxWidth, maxHeight = resizeWithAspectRatio(width, height, 1920, 1080)
	}

	ops := lilliput.NewImageOps(8192)
	defer ops.Close()

	outBuf := make([]byte, 50*1024*1024)

	imageOpts := &lilliput.ImageOptions{
		FileType:      ".webp",
		Width:         maxWidth,
		Height:        maxHeight,
		ResizeMethod:  lilliput.ImageOpsResize,
		EncodeTimeout: maxEncodeTime,
		EncodeOptions: map[int]int{lilliput.WebpQuality: 30},
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

func resizeWithAspectRatio(origWidth, origHeight, maxWidth, maxHeight int) (int, int) {
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
	var cropWidth, cropHeight, cropX, cropY int
	var inputPath, outputPath string
	var doCrop bool

	rootCmd := &cobra.Command{
		Use:   "pixiedust",
		Short: "A CLI for processing images - Magically fast!",
		Run: func(cmd *cobra.Command, args []string) {
			opts := Options{}
			err := compressImage(inputPath, outputPath, opts)
			if err != nil {
				log.Fatalf("Error resizing image: %v", err)
			}

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

				err := processImage(outputPath, outputPath, opts)
				if err != nil {
					log.Fatalf("Error processing image: %v", err)
				}
			}

			fmt.Println("Image processed and saved to", outputPath)
		},
	}

	rootCmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input image path (required)")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output image path (required)")
	rootCmd.Flags().IntVar(&cropWidth, "crop-width", 0, "Crop width")
	rootCmd.Flags().IntVar(&cropHeight, "crop-height", 0, "Crop height")
	rootCmd.Flags().IntVar(&cropX, "crop-x", 0, "Crop center X coordinate")
	rootCmd.Flags().IntVar(&cropY, "crop-y", 0, "Crop center Y coordinate")
	rootCmd.Flags().BoolVar(&doCrop, "crop", false, "Enable cropping")

	rootCmd.MarkFlagRequired("input")
	rootCmd.MarkFlagRequired("output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

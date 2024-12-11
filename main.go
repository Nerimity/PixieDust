package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/discord/lilliput"
)

func main() {
	if runtime.GOOS == "windows" {
		log.Fatal("[!] Windows isn't supported by PixieDust, quitting!")
	}

	if len(os.Args) < 3 {
		log.Fatal("[!] usage: ./pixiedust <image path> <destination path>")
	}

	imagePath := os.Args[1]
	destPath := os.Args[2]
	crop := os.Args[3]
	var cropWidth, cropHeight = 0, 0
	var cropping = false

	if crop != "" {
		fmt.Println("Using crop specified.")
		dimensions := strings.Split(crop, "x")
		parsedWidth, err := strconv.Atoi(dimensions[0])

		if err != nil {
			log.Fatal("Invalid crop! Use the following format: widthxheight")
		}

		parsedHeight, err := strconv.Atoi(dimensions[1])

		if err != nil {
			log.Fatal("Invalid crop! Use the following format: widthxheight")
		}

		cropWidth = parsedWidth
		cropHeight = parsedHeight
		cropping = true
	}

	buf, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatalf("Unable to read image: %v", err)
	}

	decoder, err := lilliput.NewDecoder(buf)
	if err != nil {
		log.Fatalf("Error decoding image: %v", err)
	}
	defer decoder.Close()

	header, err := decoder.Header()
	if err != nil {
		log.Fatalf("Error getting image header: %v", err)
	}

	width, height := header.Width(), header.Height()
	var maxWidth, maxHeight int

	maxEncodeTime, err := time.ParseDuration("30s")
	if err != nil {
		log.Fatalf("How did we get here? (line 44 duration parse failed! %v)", err)
	}

	if !cropping {
		if header.IsAnimated() {
			maxWidth, maxHeight = resizeWithAspectRatio(width, height, 800, 600)
		} else {
			maxWidth, maxHeight = resizeWithAspectRatio(width, height, 1920, 1080)
		}
	}

	ops := lilliput.NewImageOps(8192)
	defer ops.Close()

	outBuf := make([]byte, 50*1024*1024)

	var opts *lilliput.ImageOptions

	if cropping {
		opts = &lilliput.ImageOptions{
			FileType:      ".webp",
			Width:         cropWidth,
			Height:        cropHeight,
			ResizeMethod:  lilliput.ImageOpsFit,
			EncodeTimeout: maxEncodeTime,
			EncodeOptions: map[int]int{lilliput.WebpQuality: 30},
		}
	} else {
		opts = &lilliput.ImageOptions{
			FileType:      ".webp",
			Width:         maxWidth,
			Height:        maxHeight,
			ResizeMethod:  lilliput.ImageOpsResize,
			EncodeTimeout: maxEncodeTime,
			EncodeOptions: map[int]int{lilliput.WebpQuality: 30},
		}
	}

	output, err := ops.Transform(decoder, opts, outBuf)
	if err != nil {
		log.Fatalf("Error transforming image: %v", err)
	}

	err = os.WriteFile(destPath, output, 0644)
	if err != nil {
		log.Fatalf("Unable to write output file: %v", err)
	}

	fmt.Println("Image processed and saved to", destPath)
}

// Helper function to calculate new dimensions while maintaining aspect ratio
func resizeWithAspectRatio(origWidth, origHeight, maxWidth, maxHeight int) (int, int) {
	if origWidth <= maxWidth && origHeight <= maxHeight {
		return origWidth, origHeight
	}

	ratio := float64(origWidth) / float64(origHeight)
	if maxWidth/int(ratio) > maxHeight {
		return int(float64(maxHeight) * ratio), maxHeight
	}
	return maxWidth, int(float64(maxWidth) / ratio)
}

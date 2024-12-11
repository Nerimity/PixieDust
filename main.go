package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
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

	maxWidth, maxHeight = resizeWithAspectRatio(width, height, 1920, 1080)

	ops := lilliput.NewImageOps(8192)
	defer ops.Close()

	outBuf := make([]byte, 50*1024*1024)

	opts := &lilliput.ImageOptions{
		FileType:      ".webp",
		Width:         maxWidth,
		Height:        maxHeight,
		ResizeMethod:  lilliput.ImageOpsResize,
		EncodeTimeout: maxEncodeTime,
		EncodeOptions: map[int]int{lilliput.WebpQuality: 50},
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

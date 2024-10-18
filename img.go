package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path"
	"strings"

	"github.com/nfnt/resize"
)

var limit chan bool

func init() {
	limit = make(chan bool, 4)
}

// sizeCopy returns a path to the resized copy of the image at origPath.
// The image is resized proportionally so that its width and height do not
// exceed maxWidth and maxHeight.
func sizeCopy(cacheDir, origPath string, maxWidth, maxHeight int) (string, error) {
	limit <- true
	defer func() {
		<-limit
	}()
	copyPath := fmt.Sprintf("%s/%x-%dx%d.jpg", cacheDir, sha1.Sum([]byte(origPath)), maxWidth, maxHeight)

	// If the copy already exists, return.
	f0, err := os.Open(copyPath)
	if err == nil {
		f0.Close()
		return copyPath, nil
	}

	// If open failed for some other reason than "not exists", there's a problem.
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	log.Println("making "+copyPath, "from", origPath)
	img, err := imgOpen(origPath)
	if err != nil {
		return "", err
	}
	w := float32(img.Bounds().Dx())
	h := float32(img.Bounds().Dy())
	if h > float32(maxHeight) {
		w = w / h * float32(maxHeight)
		h = float32(maxHeight)
	}
	if w > float32(maxWidth) {
		h = h / w * float32(maxWidth)
		w = float32(maxWidth)
	}
	copy := resize.Resize(uint(w), uint(h), img, resize.Lanczos3)
	f2, err := os.Create(copyPath)
	if err != nil {
		panic(err)
	}
	defer f2.Close()
	err = jpeg.Encode(f2, copy, nil)
	return copyPath, err
}

func imgOpen(p string) (image.Image, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch strings.ToLower(path.Ext(p)) {
	case ".jpg":
		return jpeg.Decode(f)
	case ".gif":
		return gif.Decode(f)
	case ".png":
		return png.Decode(f)
	}
	return nil, fmt.Errorf("unknown image extension: %s", path.Ext(p))
}

func isImageExt(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".jpg" || ext == ".gif" || ext == ".png"
}

package fileHandlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/disintegration/imaging"
)

var mutex sync.Mutex

func Encode(inputBytes []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(inputBytes))
	if err != nil {
		return nil, err
	}

	// if height is larger than width, crop height to same size as width,
	// else if width is larger than height, crop width to the same size as height
	if img.Bounds().Dy() > img.Bounds().Dx() {
		img = imaging.CropCenter(img, img.Bounds().Dx(), img.Bounds().Dx())
	} else if img.Bounds().Dx() > img.Bounds().Dy() {
		img = imaging.CropCenter(img, img.Bounds().Dy(), img.Bounds().Dy())
	}

	if img.Bounds().Dx() != img.Bounds().Dy() {
		return nil, fmt.Errorf("processed avatar didn't end up being in square dimension, it's: [%dx%d]", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// resize to 256x256 width if larger
	if img.Bounds().Dx() > 256 || img.Bounds().Dy() > 256 {
		img = imaging.Resize(img, 256, 256, imaging.Lanczos)
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 50})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func HandleAvatarPicture(r *http.Request) (string, error) {
	// parse formfile
	picFormFile, _, err := r.FormFile("picture")
	if err != nil {
		return "", err
	}
	defer func() {
		err := picFormFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	// read bytes from received avatar pic
	inputBytes, err := io.ReadAll(picFormFile)
	if err != nil {
		return "", err
	}

	// encode into jpg
	resultBytes, err := Encode(inputBytes)
	if err != nil {
		return "", err
	}

	// use the hash for filename
	hash := sha256.Sum256(resultBytes)

	// construct the full path for saving
	fileName := hex.EncodeToString(hash[:]) + ".jpg"
	folderPath := filepath.Join(".", "public", "avatars")
	fullPath := filepath.Join(folderPath, fileName)

	mutex.Lock()
	defer mutex.Unlock()

	// make folders if they don't exist yet
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return "", nil
	}

	// don't overwrite if file already exists
	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		err = os.WriteFile(fullPath, resultBytes, 0644)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return fileName, nil
}

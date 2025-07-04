package fileHandlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

func HandleAvatarPicture(r *http.Request) (string, error) {
	// parse formfile
	picFormFile, _, err := r.FormFile("picture")
	if err != nil {
		return "", err
	}
	defer picFormFile.Close()

	// read bytes from received avatar pic
	imgBytes, err := io.ReadAll(picFormFile)
	if err != nil {
		return "", err
	}

	// decode
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", err
	}

	// check if picture is too small
	if img.Bounds().Dx() < 64 || img.Bounds().Dy() < 64 {
		return "", errors.New("picture is too small, minimum 64x64")
	}

	// check if picture is either too wide or too tall
	widthRatio := float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
	heightRatio := float64(img.Bounds().Dy()) / float64(img.Bounds().Dx())
	if widthRatio > 2 {
		return "", errors.New("picture is too wide, must be less than 1:2 ratio")
	} else if heightRatio > 2 {
		return "", errors.New("picture is too tall, must be less than 1:2 ratio")
	}

	// if height is larger than width, crop height to same size as width,
	// else if width is larger than height, crop width to the same size as height
	if img.Bounds().Dy() > img.Bounds().Dx() {
		img = imaging.CropCenter(img, img.Bounds().Dx(), img.Bounds().Dx())
	} else if img.Bounds().Dx() > img.Bounds().Dy() {
		img = imaging.CropCenter(img, img.Bounds().Dy(), img.Bounds().Dy())
	}

	// check if picture is in square dimension,
	// this should never be wrong as it's been cropped previously
	if img.Bounds().Dx() != img.Bounds().Dy() {
		return "", errors.New("picture isn't square size")
	}

	// resize to 256px width if wider or taller
	if img.Bounds().Dx() > 256 && img.Bounds().Dy() > 256 {
		img = imaging.Resize(img, 256, 256, imaging.Lanczos)
	}

	// recompress into jpg
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(buf.Bytes())
	fileName := hex.EncodeToString(hash[:]) + ".jpg"
	folderPath := filepath.Join(".", "public", "avatars")
	fullPath := filepath.Join(folderPath, fileName)

	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return "", nil
	}

	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		err = os.WriteFile(fullPath, buf.Bytes(), 0644)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	// no error
	return fileName, nil
}

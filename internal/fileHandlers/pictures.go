package fileHandlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var mutex sync.Mutex

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

	cmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0",
		"-vf", "crop=min(iw\\,ih):min(iw\\,ih):(iw-min(iw\\,ih))/2:(ih-min(iw\\,ih))/2,scale=256:256",
		"-vframes", "1",
		"-c:v", "libwebp",
		"-quality", "50",
		"-preset", "default",
		"-f", "webp",
		"pipe:1",
	)

	// print ffmpeg result
	// cmd.Stderr = os.Stderr

	// this will send the input picture bytes to ffmpeg
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	// this will store the converted image result
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// start the command
	err = cmd.Start()
	if err != nil {
		return "", err
	}

	// send the input bytes
	_, err = stdin.Write(inputBytes)
	if err != nil {
		return "", err
	}

	err = stdin.Close()
	if err != nil {
		return "", err
	}

	// wait for it to finish
	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	// read the converted image bytes back
	resultBytes := stdout.Bytes()

	// use the hash for filename
	hash := sha256.Sum256(resultBytes)

	// construct the full path for saving
	fileName := hex.EncodeToString(hash[:]) + ".webp"
	folderPath := filepath.Join(".", "public", "avatars")
	fullPath := filepath.Join(folderPath, fileName)

	mutex.Lock()
	defer mutex.Unlock()

	// make folders if they don't exist yet
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return "", nil
	}

	// check if avatar with same hash exists already
	_, err = os.Stat(fullPath)
	// if it doesn't exist, write it
	if os.IsNotExist(err) {
		err = os.WriteFile(fullPath, resultBytes, 0644)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	// no error
	return fileName, nil
}

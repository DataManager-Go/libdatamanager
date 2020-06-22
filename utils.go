package libdatamanager

import (
	"archive/tar"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func encodeBase64(b []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(b))
}

func decodeBase64(b []byte) []byte {
	data, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		fmt.Println("Error: Bad Key!")
		os.Exit(1)
	}
	return data
}

func archive(src string, buf io.Writer) error {
	maxErrors := 10

	tw := tar.NewWriter(buf)
	buff := make([]byte, 1024*1024)
	// baseDir := getBaseDir(src)

	errChan := make(chan error, maxErrors)

	// walk through every file in the folder
	go func() {
		filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
			if len(file) < len(src)+1 {
				return nil
			}

			// Follow link
			var link string
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				if link, err = os.Readlink(file); err != nil {
					errChan <- err
					return nil
				}
			}

			// Generate tar header
			header, err := tar.FileInfoHeader(fi, link)
			if err != nil {
				errChan <- err
				return nil
			}

			// Set filename
			header.Name = filepath.Join(src, strings.TrimPrefix(file, src))
			//header.Name = filepath.ToSlash(file)

			// write header
			if err := tw.WriteHeader(header); err != nil {
				errChan <- err
				return nil
			}

			// Nothing more to do for non-regular
			if !fi.Mode().IsRegular() {
				return nil
			}

			// can only write file-
			// contents to archives
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					errChan <- err
					return nil
				}

				if _, err := io.CopyBuffer(tw, data, buff); err != nil {
					errChan <- err
					return nil
				}

				data.Close()
			}

			return nil
		})

		close(errChan)
	}()

	errCounter := 0
	for err := range errChan {
		if errCounter >= maxErrors {
			return errors.New("Too many errors")
		}

		fmt.Println(err)
		errCounter++
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}

	return nil
}

// Get Base dir without last dir
func getBaseDir(dir string) string {
	if strings.HasSuffix(dir, string(filepath.Separator)) {
		dir = dir[:len(dir)-1]
	}

	dir = filepath.Dir(dir)

	if !strings.HasSuffix(dir, string(filepath.Separator)) {
		dir += string(filepath.Separator)
	}

	return dir
}

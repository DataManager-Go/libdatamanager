package libdatamanager

import (
	"archive/tar"
	"encoding/base64"
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
	tw := tar.NewWriter(buf)
	buff := make([]byte, 1024*1024)

	baseDir := getBaseDir(src)

	// walk through every file in the folder
	filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if len(file) < len(src)+1 {
			return nil
		}

		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)
		filen := file[len(src)+1:]
		header.Name = filepath.ToSlash(filen)

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}

			if _, err := io.CopyBuffer(tw, data, buff); err != nil {
				return err
			}
		}

		return nil
	})

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}

	return nil
}

func getBaseDir(dir string) string {
	if strings.HasSuffix(dir, string(filepath.Separator)) {
		dir = dir[:len(dir)-1]
	}

	return filepath.Dir(dir)
}

package libdatamanager

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Return byte slice with base64 encoded file content
func fileToBase64(filename string, fh *os.File) ([]byte, error) {
	s, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	src := make([]byte, s.Size())
	_, err = fh.Read(src)
	if err != nil {
		return nil, err
	}

	return encodeBase64(src), nil
}

// EncryptionCiphers supported encryption chipers
var EncryptionCiphers = []string{
	"aes",
}

// ChiperToInt cipter to int
func ChiperToInt(c string) int32 {
	c = strings.ToLower(c)
	for i, ec := range EncryptionCiphers {
		if c == strings.ToLower(ec) {
			return int32(i) + 1
		}
	}

	return 0
}

// EncryptionIValid return true if encryption i is valid
func EncryptionIValid(i int32) bool {
	if i-1 < 0 || i-1 >= int32(len(EncryptionCiphers)) {
		return false
	}

	return true
}

// ChiperToString cipter to int
func ChiperToString(i int32) string {
	if !EncryptionIValid(i) {
		return ""
	}

	return EncryptionCiphers[i-1]
}

// IsValidCipher return true if given cipher is valid
func IsValidCipher(c string) bool {
	c = strings.ToLower(c)
	for _, ec := range EncryptionCiphers {
		if strings.ToLower(ec) == c {
			return true
		}
	}

	return false
}

func respToDecrypted(resp *http.Response, encryptionKey string) (io.Reader, error) {
	var reader io.Reader

	key := []byte(encryptionKey)
	if len(key) == 0 && len(resp.Header.Get(HeaderEncryption)) > 0 {
		fmt.Println("Error: file is encrypted but no key was given. To ignore this use --no-decrypt")
		os.Exit(1)
	}

	switch resp.Header.Get(HeaderEncryption) {
	case EncryptionCiphers[0]:
		{
			// AES

			// Read response
			text, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			// Create Cipher
			block, err := aes.NewCipher(key)
			if err != nil {
				panic(err)
			}

			// Validate text length
			if len(text) < aes.BlockSize {
				fmt.Printf("Error!\n")
				os.Exit(0)
			}

			iv := text[:aes.BlockSize]
			text = text[aes.BlockSize:]

			// Decrypt
			cfb := cipher.NewCFBDecrypter(block, iv)
			cfb.XORKeyStream(text, text)

			reader = bytes.NewReader(decodeBase64(text))
		}
	case "":
		{
			reader = resp.Body
		}
	default:
		{
			return nil, errors.New("Cipher not supported")
		}
	}

	return reader, nil
}

// returns a reader to the correct source of data
func getFileEncrypter(filename string, fh *os.File, encryption, encryptionKey string) (io.Reader, int64, error) {
	var reader io.Reader
	var ln int64
	switch encryption {
	case EncryptionCiphers[0]:
		{
			// AES
			block, err := aes.NewCipher([]byte(encryptionKey))
			if err != nil {
				return nil, 0, err
			}

			// Get file content
			b, err := fileToBase64(filename, fh)
			if err != nil {
				return nil, 0, err
			}

			// Set Ciphertext 0->16 to Iv
			ciphertext := make([]byte, aes.BlockSize+len(b))
			iv := ciphertext[:aes.BlockSize]
			if _, err := io.ReadFull(rand.Reader, iv); err != nil {
				return nil, 0, err
			}

			// Encrypt file
			cfb := cipher.NewCFBEncrypter(block, iv)
			cfb.XORKeyStream(ciphertext[aes.BlockSize:], b)

			// Set reader to reader from bytes
			reader = bytes.NewReader(ciphertext)
			ln = int64(len(ciphertext))
		}
	case "":
		{
			// Set reader to reader of file
			reader = fh
		}
	default:
		{
			// Return error if cipher is not implemented
			return nil, 0, errors.New("cipher not supported")
		}
	}

	return reader, ln, nil
}

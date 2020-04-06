package libdatamanager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
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

// Encrypt encrypts input stream and writes it to out
func Encrypt(in io.Reader, out io.Writer, keyAes, buff []byte) (err error) {
	iv := make([]byte, 16)
	_, err = rand.Read(iv)
	if err != nil {
		return err
	}

	aes, err := aes.NewCipher(keyAes)
	if err != nil {
		return err
	}

	ctr := cipher.NewCTR(aes, iv)
	out.Write(iv)

	for {
		n, err := in.Read(buff)
		if err != nil && err != io.EOF {
			return err
		}

		if n != 0 {
			outBuf := make([]byte, n)
			ctr.XORKeyStream(outBuf, buff[:n])
			out.Write(outBuf)
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}

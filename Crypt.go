package libdatamanager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"strings"
)

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

// EncryptAES encrypts input stream and writes it to out
func EncryptAES(in io.Reader, out io.Writer, keyAes, buff []byte, cancel chan bool) (err error) {
	iv := make([]byte, 16)

	// Create random iv
	_, err = rand.Read(iv)
	if err != nil {
		return err
	}

	aes, err := aes.NewCipher(keyAes)
	if err != nil {
		return err
	}

	ctr := cipher.NewCTR(aes, iv)

	// First 16 bytes of ciphertext will
	// be the iv, so write it!
	out.Write(iv)

	// Encrypt file
	for {
		// Stop on cancel
		select {
		case <-cancel:
			return ErrCancelled
		default:
		}

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

// DecryptAES decrypt stuff
func DecryptAES(in io.Reader, out, hashwriter *io.Writer, keyAes, buff []byte, cancelChan chan bool) (err error) {
	// Iv is always 16 bytes
	iv := make([]byte, 16)

	aes, err := aes.NewCipher(keyAes)
	if err != nil {
		return err
	}

	// Read first 16 bytes to iv
	n, err := in.Read(iv)
	if err != nil {
		return err
	}

	// If not 16 bytes were written, there
	// is a big problem
	if n != 16 {
		return errors.New("reading aes iv")
	}

	// Write iv to hasher cause the servers
	// hash is built on it too
	if hashwriter != nil {
		(*hashwriter).Write(iv)
	}
	ctr := cipher.NewCTR(aes, iv)

	// Decrypt using xor keystream the
	// cipher input text is written to the
	// hashwriter since the servers has no
	// key and built it's hash using the
	// encrypted text
	for {
		// return on cancel
		select {
		case <-cancelChan:
			return ErrCancelled
		default:
		}

		n, err := in.Read(buff)
		if err != nil && err != io.EOF {
			return err
		}

		if n != 0 {
			outBuf := make([]byte, n)
			ctr.XORKeyStream(outBuf, buff[:n])

			(*out).Write(outBuf)

			if hashwriter != nil {
				(*hashwriter).Write(buff[:n])
			}
		}

		// Treat eof as stop condition, not as
		// an error
		if err == io.EOF {
			break
		}
	}

	return nil
}

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

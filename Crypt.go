package libdatamanager

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"strings"

	"filippo.io/age"
)

// EncryptionCiphers supported encryption method
var EncryptionCiphers = map[int8]string{
	1: "aes",
	2: "age",
}

// ChiperToInt cipter to int
func ChiperToInt(c string) int8 {
	if !IsValidCipher(c) {
		return -1
	}

	c = strings.ToLower(c)
	for i, ec := range EncryptionCiphers {
		if c == strings.ToLower(ec) {
			return i
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

// Extract public key from private key file
func getPubKeyFromIdentity(b []byte) io.Reader {
	scanner := bufio.NewScanner(bytes.NewBuffer(b))

	// Search for a 'public key entry'
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "public key:") {
			return bytes.NewBufferString(strings.TrimSpace(strings.Split(scanner.Text(), ":")[1]))
		}

		if strings.HasPrefix(scanner.Text(), "age") {
			return bytes.NewBufferString(scanner.Text())
		}
	}

	return bytes.NewBuffer(b)
}

// EncryptAGE encrypts input stream and writes it to out
func EncryptAGE(out io.Writer, in io.Reader, key, buff []byte, cancel chan bool) (err error) {
	rec, err := age.ParseRecipients(getPubKeyFromIdentity(key))
	if err != nil {
		return err
	}

	enWriter, nil := age.Encrypt(out, rec...)
	if err != nil {

		return err
	}

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
			_, err := enWriter.Write(buff[:n])
			if err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		}
	}

	enWriter.Close()

	return nil
}

// EncryptAES encrypts input stream and writes it to out
func EncryptAES(out io.Writer, in io.Reader, keyAes, buff []byte, cancel chan bool) (err error) {
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
			// TODO can we outsource this?
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

// DecryptAGE decrypt stuff
func DecryptAGE(in io.Reader, out, hashwriter io.Writer, key, buff []byte, cancelChan chan bool) (err error) {
	id, err := age.ParseIdentities(bytes.NewBuffer(key))
	if err != nil {
		return err
	}

	decReader, err := age.Decrypt(in, id...)
	if err != nil {
		return err
	}

	for {
		// return on cancel
		select {
		case <-cancelChan:
			return ErrCancelled
		default:
		}

		n, err := decReader.Read(buff)
		if err != nil && err != io.EOF {
			return err
		}

		if n != 0 {
			_, err = out.Write(buff[:n])
			if err != nil {
				return err
			}

			if hashwriter != nil {
				hashwriter.Write(buff[:n])
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
			// TODO can we outsource this?
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

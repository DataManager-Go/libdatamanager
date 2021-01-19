package libdatamanager

import (
	"encoding/hex"
	"errors"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	gzip "github.com/klauspost/pgzip"
)

var (
	// ErrChecksumNotMatch error if the checksum of the downloaded
	// file doesn't match with the checksum of the remote file
	ErrChecksumNotMatch = errors.New("generated checksum not match")

	// ErrCipherNotSupported if cipher is not supported
	ErrCipherNotSupported = errors.New("cipher not supported")

	// ErrFileEncrypted error if no key was given and nodecrypt is false
	ErrFileEncrypted = errors.New("file is encrypted but no key was given")
)

// FileDownloadRequest request for downloading a file
type FileDownloadRequest struct {
	LibDM
	ID             uint
	Name           string
	Namespace      string
	Decrypt        bool
	Key            []byte
	Buffersize     int
	ignoreChecksum bool
	CancelDownload chan bool
	WriterProxy    WriterProxy
	ReaderProxy    ReaderProxy
}

// NewFileRequest create a new filerequest
func (libdm LibDM) NewFileRequest(id uint, name, namespace string) *FileDownloadRequest {
	return &FileDownloadRequest{
		LibDM:      libdm,
		Buffersize: DefaultBuffersize,
		Decrypt:    true,
		Name:       name,
		Namespace:  namespace,
		ID:         id,
	}
}

// NewFileRequestByName create a new filerequest by name
func (libdm LibDM) NewFileRequestByName(name, namespace string) *FileDownloadRequest {
	return &FileDownloadRequest{
		LibDM:      libdm,
		Decrypt:    true,
		Name:       name,
		Namespace:  namespace,
		Buffersize: DefaultBuffersize,
	}
}

// NewFileRequestByID create a new filerequest by file id
func (libdm LibDM) NewFileRequestByID(fileID uint) *FileDownloadRequest {
	return &FileDownloadRequest{
		LibDM:      libdm,
		Decrypt:    true,
		ID:         fileID,
		Buffersize: DefaultBuffersize,
	}
}

// GetWriterProxy returns proxywriter of request
func (fileRequest *FileDownloadRequest) GetWriterProxy() WriterProxy {
	if fileRequest.WriterProxy == nil {
		return NoProxyWriter
	}

	return fileRequest.WriterProxy
}

// GetReaderProxy of request
func (fileRequest *FileDownloadRequest) GetReaderProxy() ReaderProxy {
	if fileRequest.ReaderProxy == nil {
		return NoProxyReader
	}

	return fileRequest.ReaderProxy
}

// GetBuffersize gets the buffersize
func (fileRequest *FileDownloadRequest) GetBuffersize() int {
	if fileRequest.Buffersize <= 0 {
		return DefaultBuffersize
	}
	return fileRequest.Buffersize
}

// IgnoreChecksum ignores the checksum
func (fileRequest *FileDownloadRequest) IgnoreChecksum() *FileDownloadRequest {
	fileRequest.ignoreChecksum = true
	return fileRequest
}

// NoDecrypt don't decrypt file while downloading
func (fileRequest *FileDownloadRequest) NoDecrypt() *FileDownloadRequest {
	fileRequest.Decrypt = false
	return fileRequest
}

// DecryptWith sets key to decrypt file with. If key is nil, no decryption will be performed
func (fileRequest *FileDownloadRequest) DecryptWith(key []byte) *FileDownloadRequest {
	if key == nil {
		return fileRequest.NoDecrypt()
	}

	fileRequest.Key = key
	return fileRequest
}

// Do requests a filedownload and returns the response
// The response body must be closed
func (fileRequest *FileDownloadRequest) Do() (*FileDownloadResponse, error) {
	resp, err := fileRequest.NewRequest(EPFileGet, &FileRequest{
		Name:   fileRequest.Name,
		FileID: fileRequest.ID,
		Attributes: FileAttributes{
			Namespace: fileRequest.Namespace,
		},
	}).WithAuthFromConfig().DoHTTPRequest()

	// Check for error
	if err != nil {
		return nil, &ResponseErr{
			Err: err,
		}
	}

	// Check response headers
	if resp.Header.Get(HeaderStatus) == strconv.Itoa(int(ResponseError)) {
		return nil, &ResponseErr{
			Response: &RestRequestResponse{
				HTTPCode: resp.StatusCode,
				Headers:  &resp.Header,
				Message:  resp.Header.Get(HeaderStatusMessage),
				Status:   ResponseError,
			},
		}
	}

	// Get filename from headers
	serverFileName := resp.Header.Get(HeaderFileName)
	// Get file checksum from headers
	checksum := resp.Header.Get(HeaderChecksum)
	// Get encryption header
	encryption := resp.Header.Get(HeaderEncryption)
	// Get size header
	size := GetFilesizeFromDownloadRequest(resp)
	// Get FileID
	var id uint
	iid, err := strconv.ParseUint(resp.Header.Get(HeaderFileID), 10, 32)
	if err == nil {
		id = uint(iid)
	}

	// Check headers
	if len(serverFileName) == 0 {
		return nil, &ResponseErr{
			Err: ErrResponseFilenameInvalid,
		}
	}

	// Return file response
	return &FileDownloadResponse{
		Response:        resp,
		ServerChecksum:  checksum,
		Encryption:      encryption,
		ServerFileName:  serverFileName,
		Size:            size,
		DownloadRequest: fileRequest,
		FileID:          id,
	}, nil
}

// WriteToFile saves a file to the given localFilePath containing the body of the given response
func (fileresponse *FileDownloadResponse) WriteToFile(localFilePath string, fmode os.FileMode, cancelChan chan bool) error {
	// Create loal file
	f, err := os.OpenFile(localFilePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, fmode)
	defer f.Close()
	if err != nil {
		return err
	}

	// Save body to file using given proxy
	err = fileresponse.SaveTo(f, cancelChan)
	if err != nil {
		return err
	}

	// Verify checksum if not ignored
	if !fileresponse.VerifyChecksum() && !fileresponse.DownloadRequest.ignoreChecksum {
		return ErrChecksumNotMatch
	}

	// Save to file using a proxy
	return nil
}

// DownloadToFile downloads and saves a file to the given localFilePath. If the file exists, it will be overwritten
func (fileRequest *FileDownloadRequest) DownloadToFile(localFilePath string, fmode os.FileMode, appendFilename ...bool) (*FileDownloadResponse, error) {
	resp, err := fileRequest.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Response.Body.Close()

	// Append remote filename if desired
	if len(appendFilename) > 0 && appendFilename[0] {
		localFilePath = filepath.Join(localFilePath, resp.ServerFileName)
	}

	// Create loal file
	f, err := os.OpenFile(localFilePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, fmode)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	// Write to file
	err = resp.SaveTo(f, fileRequest.CancelDownload)
	if err != nil {
		return nil, err
	}

	// Verify checksum if not disabled
	if !fileRequest.ignoreChecksum && !resp.VerifyChecksum() {
		return resp, ErrChecksumNotMatch
	}

	return resp, nil
}

// FileDownloadResponse response for downloading a file
type FileDownloadResponse struct {
	Response        *http.Response
	ServerFileName  string
	LocalChecksum   string
	ServerChecksum  string
	Size            int64
	Encryption      string
	FileID          uint
	Extract         bool
	DownloadRequest *FileDownloadRequest
}

// VerifyChecksum Return if checksums are equal and not empty
func (fileresponse *FileDownloadResponse) VerifyChecksum() bool {
	return fileresponse.ServerChecksum == fileresponse.LocalChecksum && len(fileresponse.LocalChecksum) > 0
}

// SaveTo download a file and write it to the writer while
func (fileresponse *FileDownloadResponse) SaveTo(w io.Writer, cancelChan chan bool) error {
	defer fileresponse.Response.Body.Close()

	var err error
	buff := make([]byte, fileresponse.DownloadRequest.GetBuffersize())
	hash := crc32.NewIEEE()

	reader := io.TeeReader(fileresponse.DownloadRequest.GetReaderProxy()(fileresponse.Response.Body), hash)

	var gz *gzip.Reader
	// TODO let the server decide whether to
	// extract. Otherwise compressed files
	// can't be decrypted correctly!!
	if fileresponse.Extract {
		gz, err = gzip.NewReader(reader)
		if err != nil {
			return err
		}

		reader = gz
	}

	w = fileresponse.DownloadRequest.GetWriterProxy()(w)

	// If decryption is requested and required
	if fileresponse.DownloadRequest.Decrypt && len(fileresponse.Encryption) > 0 {
		// Throw error if no key was given
		if len(fileresponse.DownloadRequest.Key) == 0 {
			return ErrFileEncrypted
		}

		switch fileresponse.Encryption {
		case EncryptionCiphers[0]:
			// Decrypt aes
			err = DecryptAES(reader, &w, nil, fileresponse.DownloadRequest.Key, buff, fileresponse.DownloadRequest.CancelDownload)
		case EncryptionCiphers[1]:
			// Decrypt aes
			err = DecryptAGE(reader, w, nil, fileresponse.DownloadRequest.Key, buff, fileresponse.DownloadRequest.CancelDownload)
		default:
			return ErrCipherNotSupported
		}
	} else {
		// Use multiwriter to write to hash and file
		// at the same time
		err = cancelledCopy(w, reader, buff, cancelChan)
	}

	// Close gzipWriter
	if fileresponse.Extract {
		gz.Close()
	}

	if err != nil {
		return err
	}

	// Set local calculated checksum
	fileresponse.LocalChecksum = hex.EncodeToString(hash.Sum(nil))
	return nil
}

func cancelledCopy(writer io.Writer, f io.Reader, buf []byte, cancelChan chan bool) error {
	for {
		// Exit on cancel
		select {
		case _ = <-cancelChan:
			return ErrCancelled
		default:
		}

		n, err := f.Read(buf)
		if n > 0 {
			_, err := writer.Write(buf[:n])
			if err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}

// GetFilesizeFromDownloadRequest returns the filesize from a
// file from the response headers
func GetFilesizeFromDownloadRequest(resp *http.Response) int64 {
	// Get the header
	sizeHeader := resp.Header.Get(HeaderContentLength)

	// Validate it
	if len(sizeHeader) > 0 {
		// Parse it
		s, err := strconv.ParseInt(sizeHeader, 10, 64)
		if err == nil {
			return s
		}
	}

	return 0
}

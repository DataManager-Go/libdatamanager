package libdatamanager

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/url"
	"os"

	"github.com/JojiiOfficial/gaw"
	gzip "github.com/klauspost/pgzip"
)

// NoProxyWriter use to fill proxyWriter arg in UpdloadFile
var NoProxyWriter WriterProxy = func(w io.Writer) io.Writer {
	return w
}

// NoProxyReader use to fill proxyWriter arg in UpdloadFile
var NoProxyReader ReaderProxy = func(r io.Reader) io.Reader {
	return r
}

var (
	// ErrUnsupportedScheme error if url has an unsupported scheme
	ErrUnsupportedScheme = errors.New("Unsupported scheme")
)

// UploadRequest a uploadrequest
type UploadRequest struct {
	LibDM
	Name             string
	Publicname       string
	Public           bool
	Attribute        FileAttributes
	ReplaceFileID    uint
	ReplaceEqualName bool
	All              bool
	Encryption       int8
	EncryptionKey    []byte
	Buffersize       int
	fileSizeCallback FileSizeCallback
	ProxyWriter      WriterProxy
	ProxyReader      ReaderProxy
	Archive          bool
	Compressed       bool
}

// NewUploadRequest create a new uploadrequest
func (libdm LibDM) NewUploadRequest(name string, attributes FileAttributes) *UploadRequest {
	if attributes.Namespace == "" {
		attributes.Namespace = "default"
	}

	return &UploadRequest{
		LibDM:     libdm,
		Name:      name,
		Attribute: attributes,
	}
}

// Compress the uploaded file
func (uploadRequest *UploadRequest) Compress() *UploadRequest {
	uploadRequest.Compressed = true
	return uploadRequest
}

// SetFileSizeCallback sets the callback if the filesize is known
func (uploadRequest *UploadRequest) SetFileSizeCallback(cb FileSizeCallback) *UploadRequest {
	uploadRequest.fileSizeCallback = cb
	return uploadRequest
}

// GetReaderProxy returns proxyReader for uploadRequest
func (uploadRequest *UploadRequest) GetReaderProxy() ReaderProxy {
	if uploadRequest.ProxyReader == nil {
		return NoProxyReader
	}

	return uploadRequest.ProxyReader
}

// GetWriterProxy returns proxywriter for uploadRequest
func (uploadRequest *UploadRequest) GetWriterProxy() WriterProxy {
	if uploadRequest.ProxyWriter == nil {
		return NoProxyWriter
	}

	return uploadRequest.ProxyWriter
}

// GetBuffersize returns the buffersize
func (uploadRequest *UploadRequest) GetBuffersize() int {
	if uploadRequest.Buffersize <= 0 {
		return DefaultBuffersize
	}

	return uploadRequest.Buffersize
}

// MakePublic upload and publish a file. If publciName is empty a
// random public name will be created serverside
func (uploadRequest *UploadRequest) MakePublic(publicName string) *UploadRequest {
	uploadRequest.Public = true
	uploadRequest.Publicname = publicName
	return uploadRequest
}

// ReplaceFileWithSameName replace files with same name
func (uploadRequest *UploadRequest) ReplaceFileWithSameName() *UploadRequest {
	uploadRequest.ReplaceEqualName = true
	return uploadRequest
}

// HandleAll applies action to all files
func (uploadRequest *UploadRequest) HandleAll() *UploadRequest {
	uploadRequest.All = true
	return uploadRequest
}

// ReplaceFileByID replace a file instead creating a new one
func (uploadRequest *UploadRequest) ReplaceFileByID(fileID uint) *UploadRequest {
	uploadRequest.ReplaceFileID = fileID
	return uploadRequest
}

// Encrypted Upload a file encrypted
func (uploadRequest *UploadRequest) Encrypted(encryptionMethod int8, key []byte) *UploadRequest {
	uploadRequest.Encryption = encryptionMethod
	uploadRequest.EncryptionKey = key
	return uploadRequest
}

// BuildRequestStruct create a uploadRequset struct using Type
func (uploadRequest *UploadRequest) BuildRequestStruct(Type UploadType) *UploadRequestStruct {
	return &UploadRequestStruct{
		UploadType:        Type,
		Name:              uploadRequest.Name,
		Attributes:        uploadRequest.Attribute,
		Encryption:        uploadRequest.Encryption,
		Public:            uploadRequest.Public,
		PublicName:        uploadRequest.Publicname,
		ReplaceFileByID:   uploadRequest.ReplaceFileID,
		Archived:          uploadRequest.Archive,
		Compressed:        uploadRequest.Compressed,
		All:               uploadRequest.All,
		ReplaceEqualNames: uploadRequest.ReplaceEqualName,
	}
}

// UploadURL uploads an url
func (uploadRequest UploadRequest) UploadURL(u *url.URL) (*UploadResponse, error) {
	if len(uploadRequest.Name) == 0 {
		uploadRequest.Name = u.Hostname()
	}

	// Allow uploading http(s) urls only
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrUnsupportedScheme
	}

	// Build request payload
	request := uploadRequest.BuildRequestStruct(URLUploadType)
	request.URL = u.String()

	// Upload
	return uploadRequest.Do(nil, request, JSONContentType)
}

// UploadFromReader upload a file using r as data source
func (uploadRequest *UploadRequest) UploadFromReader(r io.Reader, size int64, uploadDone chan string, cancel chan bool) (*UploadResponse, error) {
	// Build request and body
	request := uploadRequest.BuildRequestStruct(FileUploadType)
	body, contenttype, size := uploadRequest.UploadBodyBuilder(r, size, uploadDone, cancel)

	if body == nil {
		return nil, fmt.Errorf("body is nil")
	}

	// Run filesize callback if set
	if uploadRequest.fileSizeCallback != nil {
		uploadRequest.fileSizeCallback(size)
	}

	resp, err := uploadRequest.Do(body, request, ContentType(contenttype))
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UploadFile uploads the given file to the server
func (uploadRequest *UploadRequest) UploadFile(f *os.File, uploadDone chan string, cancel chan bool) (*UploadResponse, error) {
	// Check if file exists and use
	// its size to provide a relyable
	// upload filesize
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// Upload from file using it's io.Reader
	return uploadRequest.UploadFromReader(f, fi.Size(), uploadDone, cancel)
}

// UploadArchivedFolder uploads the given folder to the server
func (uploadRequest *UploadRequest) UploadArchivedFolder(uri string, uploadDone chan string, cancel chan bool) (*UploadResponse, error) {
	uploadRequest.Archive = true

	// Use size of all files in dir as
	// full upload size
	size, err := gaw.GetDirSize(uri)
	if err != nil {
		return nil, err
	}

	errChan := make(chan error, 1)
	doneChan := make(chan struct{}, 1)

	pr, pw := io.Pipe()
	defer pr.Close()

	go func() {
		// Compress dir
		if err := archive(uri, pw); err != nil {
			errChan <- err
		}
		defer pw.Close()
	}()

	var resp *UploadResponse
	go func() {
		// Upload from compress reader
		resp, err = uploadRequest.UploadFromReader(pr, size, uploadDone, cancel)
		if err != nil {
			errChan <- err
		} else {
			doneChan <- struct{}{}
		}
	}()

	select {
	case err := <-errChan:
		go func() {
			uploadDone <- ""
		}()
		return nil, err
	case <-doneChan:
		return resp, nil
	}
}

// Do does the final upload http request and uploads the src
func (uploadRequest *UploadRequest) Do(body io.Reader, payload interface{}, contentType ContentType) (*UploadResponse, error) {
	// Make json header content
	rbody, err := json.Marshal(payload)
	if err != nil {
		return nil, &ResponseErr{
			Err: err,
		}
	}

	// Do request
	var resStruct UploadResponse
	response, err := uploadRequest.NewRequest(EPFileUpload, body).
		WithMethod(PUT).
		// Negate the 'Compressed' field, to tell the server
		// not to decompress the stream
		WithCompression(!uploadRequest.Compressed).
		WithAuth(uploadRequest.Config.GetBearerAuth()).WithHeader(HeaderRequest, base64.StdEncoding.EncodeToString(rbody)).
		WithRequestType(RawRequestType).
		WithContentType(contentType).
		Do(&resStruct)

	if err != nil || response.Status == ResponseError {
		return nil, NewErrorFromResponse(response, err)
	}

	return &resStruct, err
}

// UploadBodyBuilder build the body for the upload file request
func (uploadRequest *UploadRequest) UploadBodyBuilder(reader io.Reader, inpSize int64, doneChan chan string, cancel chan bool) (r *io.PipeReader, contentType string, size int64) {
	// Apply readerproxy
	reader = uploadRequest.GetReaderProxy()(reader)
	var err error

	// Don't calculate a size if inputsize
	// is empty to prevent returning an inalid size
	if inpSize > 0 {
		// Set filesize
		switch uploadRequest.Encryption {
		case 1:
			size = inpSize + aes.BlockSize
		case 2:
			size = inpSize
		case 0:
			size = inpSize
		default:
			return nil, "", -1
		}
	}

	r, pW := io.Pipe()

	go func() {
		// Create hashobject and use a multiwriter to
		// write to the part and the hash at thes
		var writer io.Writer
		var gzipw *gzip.Writer
		hash := crc32.NewIEEE()

		if uploadRequest.Compressed {
			// Server uses gzip stream to build chehcksum
			// Write zip stream into hash writer as well
			gzipw = gzip.NewWriter(io.MultiWriter((uploadRequest.GetWriterProxy()(pW)), hash))
			writer = gzipw
		} else {
			// Server uses raw stream to build chehcksum, so put
			// the hash writer separate
			gzipw = gzip.NewWriter(uploadRequest.GetWriterProxy()(pW))
			writer = io.MultiWriter(gzipw, hash)
		}

		buf := make([]byte, uploadRequest.GetBuffersize())

		// Copy from input reader to writer using
		// to support encryption
		switch uploadRequest.Encryption {
		case 1:
			err = EncryptAES(writer, reader, uploadRequest.EncryptionKey, buf, cancel)
		case 2:
			err = EncryptAGE(writer, reader, uploadRequest.EncryptionKey, buf, cancel)
		case 0:
			err = cancelledCopy(writer, reader, buf, cancel)
		}

		var hsh string
		if uploadRequest.Compressed {
			// No server-side decompression will be executed
			// so append hash to the end after full gzip stream
			gzipw.Close()
			hsh = hex.EncodeToString(hash.Sum(nil))
			pW.Write([]byte(hsh))
		} else {
			// Server wil decompress data
			// so add hash to the end and gzip
			// it as well
			hsh = hex.EncodeToString(hash.Sum(nil))
			writer.Write([]byte(hsh))
			gzipw.Close()
		}

		// Close everything and write into doneChan

		if err != nil {
			if err != ErrCancelled {
				pW.CloseWithError(err)
				doneChan <- ""
			} else {
				pW.Close()
				doneChan <- "cancelled"
			}
		} else {
			pW.Close()
			doneChan <- hsh
		}
	}()

	return
}

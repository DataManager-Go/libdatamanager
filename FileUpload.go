package libdatamanager

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash/crc32"
	"io"
	"mime/multipart"
	"net/url"
	"os"
)

// NoProxyWriter use to fill proxyWriter arg in UpdloadFile
var NoProxyWriter WriterProxy = func(w io.Writer) io.Writer {
	return w
}

var (
	// ErrUnsupportedScheme error if url has an unsupported scheme
	ErrUnsupportedScheme = errors.New("Unsupported scheme")
)

// Boundary boundary for the part
const (
	Boundary = "MachliJalKiRaniHaiJeevanUskaPaaniHai"
)

// UploadRequest a uploadrequest
type UploadRequest struct {
	LibDM
	Name             string
	Publicname       string
	Public           bool
	Attribute        FileAttributes
	ReplaceFileID    uint
	Encryption       string
	EncryptionKey    []byte
	Buffersize       int
	fileSizeCallback FileSizeCallback
	ProxyWriter      WriterProxy
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

// SetFileSizeCallback sets the callback if the filesize is known
func (uploadRequest *UploadRequest) SetFileSizeCallback(cb FileSizeCallback) *UploadRequest {
	uploadRequest.fileSizeCallback = cb
	return uploadRequest
}

// GetProxy returns proxywriter for uploadRequest
func (uploadRequest *UploadRequest) GetProxy() WriterProxy {
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

// ReplaceFile replace a file instead creating a new one
func (uploadRequest *UploadRequest) ReplaceFile(fileID uint) *UploadRequest {
	uploadRequest.ReplaceFileID = fileID
	return uploadRequest
}

// Encrypted Upload a file encrypted
func (uploadRequest *UploadRequest) Encrypted(encryptionMethod string, key []byte) *UploadRequest {
	uploadRequest.Encryption = encryptionMethod
	uploadRequest.EncryptionKey = key
	return uploadRequest
}

// BuildRequestStruct create a uploadRequset struct using Type
func (uploadRequest *UploadRequest) BuildRequestStruct(Type UploadType) *UploadRequestStruct {
	return &UploadRequestStruct{
		Name:        uploadRequest.Name,
		Attributes:  uploadRequest.Attribute,
		Encryption:  uploadRequest.Encryption,
		Public:      uploadRequest.Public,
		PublicName:  uploadRequest.Publicname,
		ReplaceFile: uploadRequest.ReplaceFileID,
		UploadType:  Type,
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
	request.Size = size

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
	// Don't calculate a size if inputsize
	// is empty to prevent returning an inalid size
	if inpSize > 0 {
		// Set filesize
		switch uploadRequest.Encryption {
		case EncryptionCiphers[0]:
			size = inpSize + aes.BlockSize
		case "":
			size = inpSize
		default:
			return nil, "", -1
		}

		// Add boundary len cause this will be
		// written as well
		size += int64(len(Boundary))
	}

	r, pW := io.Pipe()

	// Create multipart
	multipartW := multipart.NewWriter(pW)
	multipartW.SetBoundary(Boundary)
	contentType = multipartW.FormDataContentType()

	go func() {
		partW, err := multipartW.CreateFormFile("fakefield", "file")
		if err != nil {
			pW.CloseWithError(err)
			doneChan <- ""
			return
		}

		// Create hashobject and use a multiwriter to
		// write to the part and the hash at thes
		hash := crc32.NewIEEE()
		writer := io.MultiWriter((uploadRequest.GetProxy()(partW)), hash)

		buf := make([]byte, uploadRequest.GetBuffersize())

		// Copy from input reader to writer using
		// to support encryption
		switch uploadRequest.Encryption {
		case EncryptionCiphers[0]:
			err = EncryptAES(reader, writer, uploadRequest.EncryptionKey, buf, cancel)
		case "":
			err = cancelledCopy(writer, reader, buf, cancel)
		}

		// Close everything and write into doneChan
		multipartW.Close()
		if err != nil {
			pW.CloseWithError(err)
			doneChan <- ""
		} else {
			pW.Close()
			doneChan <- hex.EncodeToString(hash.Sum(nil))
		}
	}()

	return
}

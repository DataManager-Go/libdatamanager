package libdatamanager

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"hash/crc32"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// FileAttributes attributes for a file
type FileAttributes struct {
	Tags      []string `json:"tags,omitempty"`
	Groups    []string `json:"groups,omitempty"`
	Namespace string   `json:"ns"`
}

// FileUpdateItem lists changes to a file
type FileUpdateItem struct {
	IsPublic     string   `json:"ispublic,omitempty"`
	NewName      string   `json:"name,omitempty"`
	NewNamespace string   `json:"namespace,omitempty"`
	RemoveTags   []string `json:"rem_tags,omitempty"`
	RemoveGroups []string `json:"rem_groups,omitempty"`
	AddTags      []string `json:"add_tags,omitempty"`
	AddGroups    []string `json:"add_groups,omitempty"`
}

// FileResponseItem file item for file response
type FileResponseItem struct {
	ID           uint           `json:"id"`
	Size         int64          `json:"size"`
	CreationDate time.Time      `json:"creation"`
	Name         string         `json:"name"`
	IsPublic     bool           `json:"isPub"`
	PublicName   string         `json:"pubname"`
	Attributes   FileAttributes `json:"attrib"`
	Encryption   string         `json:"e"`
	Checksum     string         `json:"checksum"`
}

// FileChanges file changes for updating a file
type FileChanges struct {
	NewName                  string
	NewNamespace             string
	AddTags, AddGroups       []string
	RemoveTags, RemoveGroups []string
	SetPublic, SetPrivate    bool
}

// DeleteFile deletes the desired file(s)
func (libdm LibDM) DeleteFile(name string, id uint, all bool, attributes FileAttributes) (*IDsResponse, error) {
	var response IDsResponse

	if _, err := libdm.Request(EPFileDelete, &FileRequest{
		Name:       name,
		FileID:     id,
		All:        all,
		Attributes: attributes,
	}, &response, true); err != nil {
		return nil, err
	}

	return &response, nil
}

// ListFiles lists the files corresponding to the args
func (libdm LibDM) ListFiles(name string, id uint, allNamespaces bool, attributes FileAttributes, verbose uint8) (*FileListResponse, error) {
	var response FileListResponse

	if _, err := libdm.Request(EPFileList, &FileListRequest{
		FileID:        id,
		Name:          name,
		AllNamespaces: allNamespaces,
		Attributes:    attributes,
		OptionalParams: OptionalRequetsParameter{
			Verbose: verbose,
		},
	}, &response, true); err != nil {
		return nil, err
	}

	return &response, nil
}

// PublishFile publishs a file. If "all" is true, the response object is BulkPublishResponse. Else it is PublishResponse
func (libdm LibDM) PublishFile(name string, id uint, publicName string, all bool, attributes FileAttributes) (interface{}, error) {
	request := *NewRequest(EPFilePublish, FileRequest{
		Name:       name,
		FileID:     id,
		PublicName: publicName,
		All:        all,
		Attributes: attributes,
	}, libdm.Config).WithAuthFromConfig()

	var err error
	var response *RestRequestResponse
	var resp interface{}

	if all {
		var respData BulkPublishResponse
		response, err = request.Do(&respData)
		resp = respData
	} else {
		var respData PublishResponse
		response, err = request.Do(&respData)
		resp = respData
	}

	if err != nil || response.Status == ResponseError {
		return nil, err
	}

	return resp, nil
}

// GetFile requests a filedownload and returns the response
// -> response, serverfilename, checksum, error
// The response body must be closed
func (libdm LibDM) GetFile(name string, id uint, namespace string) (*http.Response, string, string, error) {
	resp, err := NewRequest(EPFileGet, &FileRequest{
		Name:   name,
		FileID: id,
		Attributes: FileAttributes{
			Namespace: namespace,
		},
	}, libdm.Config).WithAuthFromConfig().DoHTTPRequest()

	// Check for error
	if err != nil {
		return nil, "", "", &ResponseErr{
			Err: err,
		}
	}

	// Check response headers
	if resp.Header.Get(HeaderStatus) == strconv.Itoa(int(ResponseError)) {
		return nil, "", "", &ResponseErr{
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

	// Check headers
	if len(serverFileName) == 0 {
		return nil, "", "", &ResponseErr{
			Err: ErrResponseFilenameInvalid,
		}
	}

	return resp, serverFileName, checksum, nil
}

var (
	// ErrChecksumNotMatch error if the checksum of the downloaded
	// file doesn't match with the checksum of the remote file
	ErrChecksumNotMatch = errors.New("generated checksum not match")
)

// DownloadFile downloads and saves a file to the given localFilePath. If the file exists, it will be overwritten
func (libdm LibDM) DownloadFile(name string, id uint, namespace, localFilePath string, appendFilename ...bool) error {
	// Download file from server
	resp, name, checksum, err := libdm.GetFile(name, id, namespace)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	hash := crc32.NewIEEE()

	// Append remote filename if desired
	if len(appendFilename) > 0 && appendFilename[0] {
		localFilePath = filepath.Join(localFilePath, name)
	}

	// Create loal file
	f, err := os.Create(localFilePath)
	defer f.Close()
	if err != nil {
		return err
	}

	w := io.MultiWriter(hash, f)

	// Save file to local file
	buff := make([]byte, 10*1024)
	_, err = io.CopyBuffer(w, resp.Body, buff)
	if err != nil {
		return err
	}

	// Check if the checksums are equal, if not return an error
	if hex.EncodeToString(hash.Sum(nil)) != checksum {
		return ErrChecksumNotMatch
	}

	return nil
}

// UpdateFile updates a file on the server
func (libdm LibDM) UpdateFile(name string, id uint, namespace string, all bool, changes FileChanges) (*CountResponse, error) {
	// Set attributes
	attributes := FileAttributes{
		Namespace: namespace,
	}

	var isPublic string
	if changes.SetPublic {
		isPublic = "true"
	}
	if changes.SetPrivate {
		isPublic = "false"
	}

	// Set fileUpdates
	fileUpdates := FileUpdateItem{
		IsPublic:     isPublic,
		NewName:      changes.NewName,
		NewNamespace: changes.NewNamespace,
		RemoveTags:   changes.RemoveTags,
		RemoveGroups: changes.RemoveGroups,
		AddTags:      changes.AddTags,
		AddGroups:    changes.AddGroups,
	}

	var response CountResponse

	if _, err := libdm.Request(EPFileUpdate, &FileRequest{
		Name:       name,
		FileID:     id,
		All:        all,
		Updates:    fileUpdates,
		Attributes: attributes,
	}, &response, true); err != nil {
		return nil, err
	}

	return &response, nil
}

// NoProxyWriter use to fill proxyWriter arg in UpdloadFile
var NoProxyWriter = func(w io.Writer) io.Writer {
	return w
}

// UploadFile uploads the given file to the server and set's its affiliations
// strargs:
// [0] public name
// [1] encryption
// [2] encryptionKey
func (libdm LibDM) UploadFile(path, name string, public bool, replaceFile uint, attributes FileAttributes, proxyWriter func(w io.Writer) io.Writer, fsDetermined chan int64, done chan string, strArgs ...string) (*UploadResponse, error) {
	if replaceFile < 0 {
		replaceFile = 0
	}

	var encryption, encryptionKey, publicName string
	if len(strArgs) > 0 {
		publicName = strArgs[0]
	}
	if len(strArgs) > 1 {
		encryption = strArgs[1]
	}
	if len(strArgs) > 2 {
		encryptionKey = strArgs[2]
	}

	// Bulid request
	request := UploadRequest{
		Name:        name,
		Attributes:  attributes,
		Public:      public,
		PublicName:  publicName,
		Encryption:  encryption,
		ReplaceFile: replaceFile,
	}

	var contentType string
	var body io.Reader

	// Check for url/file
	u, err := url.Parse(path)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		request.UploadType = URLUploadType
		request.URL = path

		contentType = string(JSONContentType)
	} else {
		// Open file
		f, err := os.Open(path)
		if err != nil {
			// Write 0 into channel to prevent deatlocks
			if fsDetermined != nil {
				fsDetermined <- 0
			}
			return nil, &ResponseErr{Err: err}
		}

		// Init upload stuff
		request.UploadType = FileUploadType
		body, contentType, request.Size = FileUploader(f, proxyWriter, encryption, encryptionKey, done)
	}

	if fsDetermined != nil {
		fsDetermined <- request.Size
	}

	// Make json header content
	rbody, err := json.Marshal(request)
	if err != nil {
		return nil, &ResponseErr{
			Err: err,
		}
	}

	rBase := base64.StdEncoding.EncodeToString(rbody)

	// Do request
	var resStruct UploadResponse
	response, err := NewRequest(EPFileUpload, body, libdm.Config).
		WithMethod(PUT).
		WithAuth(libdm.Config.GetBearerAuth()).WithHeader(HeaderRequest, rBase).
		WithRequestType(RawRequestType).
		WithContentType(ContentType(contentType)).
		Do(&resStruct)

	if err != nil || response.Status == ResponseError {
		return nil, NewErrorFromResponse(response, err)
	}

	return &resStruct, nil
}

// Boundary boundary for the part
var Boundary = "MachliJalKiRaniHaiJeevanUskaPaaniHai"

// FileUploader upload a file directly
func FileUploader(f *os.File, proxyWriter func(io.Writer) io.Writer, encryption, encryptionKey string, done chan string) (r *io.PipeReader, contentType string, size int64) {
	fi, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}

	// Set filesize
	switch encryption {
	case EncryptionCiphers[0]:
		size = fi.Size() + aes.BlockSize
	case "":
		size = fi.Size()
	default:
		return nil, "", -1
	}

	// Add boundary len cause this will be
	// written as well
	size += int64(len(Boundary))

	r, pW := io.Pipe()

	// Create multipart
	multipartW := multipart.NewWriter(pW)
	multipartW.SetBoundary(Boundary)
	contentType = multipartW.FormDataContentType()

	go func() {
		partW, err := multipartW.CreateFormFile("fakefield", f.Name())
		if err != nil {
			pW.CloseWithError(err)
			done <- ""
			return
		}

		// Create hashobject and use a multiwriter to
		// write to the part and the hash at thes
		hash := crc32.NewIEEE()
		writer := io.MultiWriter(proxyWriter(partW), hash)

		buf := make([]byte, 10*1024)

		switch encryption {
		case EncryptionCiphers[0]:
			{
				err = Encrypt(f, writer, []byte(encryptionKey), buf)
			}
		case "":
			{
				err = upload(f, writer, buf)
			}
		}

		f.Close()
		multipartW.Close()

		if err != nil {
			pW.CloseWithError(err)
			done <- ""
		} else {
			pW.Close()
			done <- hex.EncodeToString(hash.Sum(nil))
		}
	}()

	return
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

func upload(f io.Reader, writer io.Writer, buf []byte) error {
	for {
		n, err := f.Read(buf)
		if n > 0 {
			_, err := writer.Write(buf[:n])
			if err != nil {
				log.Fatal(err)
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

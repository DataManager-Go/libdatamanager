package libdatamanager

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
}

type FileChanges struct {
	NewName                  string
	NewNamespace             string
	AddTags, AddGroups       []string
	RemoveTags, RemoveGroups []string
	SetPublic, SetPrivate    bool
}

// DeleteFile deletes the desired file(s)
func (libdm LibDM) DeleteFile(name string, id uint, all bool, attributes FileAttributes) (*CountResponse, error) {
	var response CountResponse

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

// DownloadFileToReader returns a readCloser for the request body == file content
// Body must be closed
func (libdm LibDM) DownloadFileToReader(name string, id uint, namespace string) (io.ReadCloser, string, error) {
	resp, err := NewRequest(EPFileGet, &FileRequest{
		Name:   name,
		FileID: id,
		Attributes: FileAttributes{
			Namespace: namespace,
		},
	}, libdm.Config).WithAuthFromConfig().DoHTTPRequest()

	// Check for error
	if err != nil {
		return nil, "", &ResponseErr{
			Err: err,
		}
	}

	// Check response headers
	if resp.Header.Get(HeaderStatus) == strconv.Itoa(int(ResponseError)) {
		return nil, "", &ResponseErr{
			Response: &RestRequestResponse{
				HTTPCode: resp.StatusCode,
				Headers:  &resp.Header,
				Message:  resp.Header.Get(HeaderStatusMessage),
				Status:   ResponseError,
			},
		}
	}

	// Get filename from response headers
	serverFileName := resp.Header.Get(HeaderFileName)

	// Check headers
	if len(serverFileName) == 0 {
		return nil, "", &ResponseErr{
			Err: ErrResponseFilenameInvalid,
		}
	}

	return resp.Body, serverFileName, nil
}

// DownloadFile downloads and saves a file to the given localFilePath. If the file exists, it will be overwritten
func (libdm LibDM) DownloadFile(name string, id uint, namespace, localFilePath string, appendFilename ...bool) error {
	// Download file from server
	rcl, name, err := libdm.DownloadFileToReader(name, id, namespace)
	if err != nil {
		return err
	}
	defer rcl.Close()

	// Append remote filename if desired
	if len(appendFilename) > 0 && appendFilename[0] {
		localFilePath = filepath.Join(localFilePath, name)
	}

	// Create loal file
	f, err := os.Create(localFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Save file to local file
	buff := make([]byte, 512)
	_, err = io.CopyBuffer(f, rcl, buff)
	if err != nil {
		return err
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

// NopNoProxyWriter use to fill proxyWriter arg in UpdloadFile
var NoProxyWriter = func(w io.Writer) io.Writer {
	return w
}

// UploadFile uploads the given file to the server and set's its affiliations
// strargs:
// [0] public name
// [1] encryption
// [2] encryptionKey
func (libdm LibDM) UploadFile(localFile, name string, public bool, replaceFile uint, attributes FileAttributes, proxyWriter func(w io.Writer) io.Writer, fsDetermined chan int64, strArgs ...string) (*UploadResponse, error) {
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
	body, contentType, request.Size = FileUploader(localFile, proxyWriter, encryption, encryptionKey)
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
		return nil, NewErrorFromResponse(response)
	}

	return &resStruct, nil
}

var Boundary = "MachliJalKiRaniHaiJeevanUskaPaaniHai"

func FileUploader(path string, proxyWriter func(io.Writer) io.Writer, encryption, encryptionKey string) (r *io.PipeReader, contentType string, size int64) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve fileSize
	fi, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}

	// Encrypt file if required
	reader, ln, err := getFileEncrypter(path, f, encryption, encryptionKey)
	if err != nil {
		log.Fatal(err)
	}

	if ln > 0 {
		size = ln
	} else {
		size = fi.Size()
	}

	r, w := io.Pipe()
	mpw := multipart.NewWriter(w)
	mpw.SetBoundary(Boundary)
	contentType = mpw.FormDataContentType()

	go func() {
		part, err := mpw.CreateFormFile("file", fi.Name())
		if err != nil {
			log.Fatal(err)
		}

		// Allow overwriting the part writer for eg. A progressbar
		part = proxyWriter(part)
		buf := make([]byte, 512)

		//_, err = io.CopyBuffer(part, reader, buf)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				break
			}
			part.Write(buf[:n])
		}

		w.Close()
		f.Close()
		mpw.Close()
	}()

	return
}

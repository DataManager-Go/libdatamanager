package libdatamanager

import (
	"io"
	"net/http"
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

const (
	// DefaultBuffersize The default buffersize for filestreams
	DefaultBuffersize = 10 * 1024
)

// WriterProxy proxy writing
type WriterProxy func(io.Writer) io.Writer

// FileSizeCallback gets called if the filesize is known
type FileSizeCallback func(int64)

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
	request := libdm.NewRequest(EPFilePublish, FileRequest{
		Name:       name,
		FileID:     id,
		PublicName: publicName,
		All:        all,
		Attributes: attributes,
	}).WithAuthFromConfig()

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

	// Do request
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

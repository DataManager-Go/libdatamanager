package libdatamanager

import (
	"io"
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
func (libdm LibDM) DownloadFileToReader(name string, id uint, namespace string) (*io.ReadCloser, string, error) {
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

	return &resp.Body, serverFileName, nil
}

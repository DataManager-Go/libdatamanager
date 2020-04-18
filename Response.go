package libdatamanager

import (
	"net/http"
)

// ResponseStatus the status of response
type ResponseStatus uint8

const (
	// ResponseError if there was an error
	ResponseError ResponseStatus = 0
	// ResponseSuccess if the response is successful
	ResponseSuccess ResponseStatus = 1
)

const (
	// HeaderStatus headername for status in response
	HeaderStatus string = "X-Response-Status"

	// HeaderStatusMessage headername for status in response
	HeaderStatusMessage string = "X-Response-Message"

	// HeaderContentType contenttype of response
	HeaderContentType string = "Content-Type"

	// HeaderFileName filename header
	HeaderFileName string = "X-Filename"

	// HeaderRequest request content
	HeaderRequest string = "Request"

	// HeaderEncryption encryption header
	HeaderEncryption string = "X-Encryption"

	// HeaderContentLength request content length
	HeaderContentLength string = "ContentLength"

	// HeaderFileID fileid header
	HeaderFileID string = "X-FileID"

	// HeaderChecksum files checksum
	HeaderChecksum string = "Checksum"
)

// LoginResponse response for login
type LoginResponse struct {
	Token     string `json:"token"`
	Namespace string `json:"ns"`
}

// RestRequestResponse the response of a rest call
type RestRequestResponse struct {
	HTTPCode int
	Status   ResponseStatus
	Message  string
	Headers  *http.Header
}

// StringResponse response containing only one string
type StringResponse struct {
	String string `json:"content"`
}

// StringSliceResponse response containing only one string slice
type StringSliceResponse struct {
	Slice []string `json:"slice"`
}

// FileListResponse response for listing files
type FileListResponse struct {
	Files []FileResponseItem
}

// UploadResponse response for uploading file
type UploadResponse struct {
	FileID         uint   `json:"fileID"`
	Filename       string `json:"filename"`
	PublicFilename string `json:"publicFilename,omitempty"`
	Checksum       string `json:"checksum"`
	FileSize       int64  `json:"size"`
	Namespace      string `json:"ns"`
}

// PublishResponse response for publishing a file
type PublishResponse struct {
	PublicFilename string `json:"pubName"`
}

// BulkPublishResponse response for publishing a file
type BulkPublishResponse struct {
	Files []UploadResponse `json:"files"`
}

// CountResponse response containing a count of changed items
type CountResponse struct {
	Count uint32 `json:"count"`
}

// IDsResponse response containing a list of ids
type IDsResponse struct {
	IDs []uint `json:"ids"`
}

// UserAttributeDataResponse response for userattribute data
type UserAttributeDataResponse struct {
	Namespace []Namespaceinfo `json:"nsData"`
}

// Namespaceinfo info for namespace
type Namespaceinfo struct {
	Name   string   `json:"ns"`
	Groups []string `json:"groups"`
}

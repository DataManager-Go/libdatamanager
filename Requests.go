package libdatamanager

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

// Method http request method
type Method string

// Requests
const (
	GET    Method = "GET"
	POST   Method = "POST"
	DELETE Method = "DELETE"
	PUT    Method = "PUT"
)

// ContentType contenttype header of request
type ContentType string

// Content types
const (
	JSONContentType ContentType = "application/json"
)

// PingRequest a ping request content
type PingRequest struct {
	Payload string
}

// Endpoint a remote url-path
type Endpoint string

// Remote endpoints
const (
	// Ping
	EPPing Endpoint = "/ping"

	// User
	EPUser      Endpoint = "/user"
	EPLogin              = EPUser + "/login"
	EPRegister           = EPUser + "/register"
	EPUserStats          = EPUser + "/stats"

	// Files
	EPFile        Endpoint = "/file"
	EPFileList             = EPFile + "s"
	EPFileUpdate           = EPFile + "/update"
	EPFileDelete           = EPFile + "/delete"
	EPFileGet              = "/download/file"
	EPFilePublish          = EPFile + "/publish"

	// Upload
	EPFileUpload Endpoint = "/upload" + EPFile

	// Attribute
	EPAttribute  Endpoint = "/attribute"
	EPAttributes Endpoint = "/attributes"
	// Tags
	EPAttributeTag = EPAttribute + "/tag"
	EPTagCreate    = EPAttributeTag + "/create"
	EPTagUpdate    = EPAttributeTag + "/update"
	EPTagDelete    = EPAttributeTag + "/delete"
	EPTags         = EPAttributeTag + "/get"
	// Group
	EPAttributeGroup = EPAttribute + "/group"
	EPGroupCreate    = EPAttributeGroup + "/create"
	EPGroupUpdate    = EPAttributeGroup + "/update"
	EPGroupDelete    = EPAttributeGroup + "/delete"
	EPGroups         = EPAttributeGroup + "/get"

	// Namespace
	EPNamespace       Endpoint = "/namespace"
	EPNamespaceCreate          = EPNamespace + "/create"
	EPNamespaceUpdate          = EPNamespace + "/update"
	EPNamespaceDelete          = EPNamespace + "/delete"
	EPNamespaceList            = EPNamespace + "s"
)

// RequestConfig configurations for requests
type RequestConfig struct {
	IgnoreCert   bool
	URL          string
	MachineID    string
	Username     string
	SessionToken string
}

// GetBearerAuth returns bearer authorization from config
func (rc RequestConfig) GetBearerAuth() Authorization {
	return Authorization{
		Type:    Bearer,
		Palyoad: rc.SessionToken,
	}
}

// Request a rest server request
type Request struct {
	RequestType           RequestType
	Endpoint              Endpoint
	Payload               interface{}
	Config                *RequestConfig
	Method                Method
	ContentType           ContentType
	Authorization         *Authorization
	Headers               map[string]string
	BenchChan             chan time.Time
	Compressed            bool
	MaxConnectionsPerHost int
}

// FileListRequest contains file info (and a file)
type FileListRequest struct {
	FileID         uint                     `json:"fid"`
	Name           string                   `json:"name"`
	AllNamespaces  bool                     `json:"allns"`
	OptionalParams OptionalRequetsParameter `json:"opt"`
	Order          string                   `json:"order,omitempty"`
	Attributes     FileAttributes           `json:"attributes"`
}

// OptionalRequetsParameter optional request parameter
type OptionalRequetsParameter struct {
	Verbose uint8 `json:"verb"`
}

// FileRequest contains data to update a file
type FileRequest struct {
	FileID     uint           `json:"fid"`
	Name       string         `json:"name,omitempty"`
	PublicName string         `json:"pubname,omitempty"`
	Updates    FileUpdateItem `json:"updates,omitempty"`
	All        bool           `json:"all"`
	Attributes FileAttributes `json:"attributes"`
}

// UpdateAttributeRequest contains data to update a tag
type UpdateAttributeRequest struct {
	Name      string `json:"name"`
	NewName   string `json:"newname"`
	Namespace string `json:"namespace"`
}

// CredentialsRequest request containing credentials
type CredentialsRequest struct {
	MachineID string `json:"mid,omitempty"`
	Username  string `json:"username"`
	Password  string `json:"pass"`
}

// NamespaceRequest namespace action request
type NamespaceRequest struct {
	Namespace string `json:"ns"`
	NewName   string `json:"newName,omitempty"`
}

// UserAttributesRequest request for getting
// namespaces and groups
type UserAttributesRequest struct {
	Mode uint `json:"m"`
}

// UploadRequestStruct contains file info (and a file)
type UploadRequestStruct struct {
	// Required fields
	UploadType UploadType `json:"type"`
	Name       string     `json:"name"`

	// Optional fields
	URL               string         `json:"url,omitempty"`
	Public            bool           `json:"pb,omitempty"`
	PublicName        string         `json:"pbname,omitempty"`
	Attributes        FileAttributes `json:"attr,omitempty"`
	Encryption        int8           `json:"e,omitempty"`
	Compressed        bool           `json:"compr,omitempty"`
	Archived          bool           `json:"arved,omitempty"`
	ReplaceFileByID   uint           `json:"r,omitempty"`
	ReplaceEqualNames bool           `json:"ren"`
	All               bool           `json:"a"`
}

// StatsRequestStruct informations about a stat-request
type StatsRequestStruct struct {
	Namespace string `json:"ns,omitempty"`
}

// UploadType type of upload
type UploadType uint8

// Available upload types
const (
	FileUploadType UploadType = iota
	URLUploadType
)

// RequestType type of request
type RequestType uint8

// Request types
const (
	JSONRequestType RequestType = iota
	RawRequestType
)

// NewRequest creates a new post request
func (limdm *LibDM) NewRequest(endpoint Endpoint, payload interface{}) *Request {
	return &Request{
		RequestType:           JSONRequestType,
		Endpoint:              endpoint,
		Payload:               payload,
		Config:                limdm.Config,
		Method:                POST,
		ContentType:           JSONContentType,
		MaxConnectionsPerHost: 1,
	}
}

// WithConnectionLimit set limit of max connectionts per host
func (request *Request) WithConnectionLimit(maxConnections int) *Request {
	request.MaxConnectionsPerHost = maxConnections
	return request
}

// WithCompression use a different method
func (request *Request) WithCompression(compression bool) *Request {
	request.Compressed = compression
	return request
}

// WithMethod use a different method
func (request *Request) WithMethod(m Method) *Request {
	request.Method = m
	return request
}

// WithRequestType use different request type
func (request *Request) WithRequestType(rType RequestType) *Request {
	request.RequestType = rType
	return request
}

// WithAuth with authorization
func (request *Request) WithAuth(a Authorization) *Request {
	request.Authorization = &a
	return request
}

// WithAuthFromConfig with authorization
func (request *Request) WithAuthFromConfig() *Request {
	auth := request.Config.GetBearerAuth()
	request.Authorization = &auth
	return request
}

// WithBenchCallback with bench
func (request *Request) WithBenchCallback(c chan time.Time) *Request {
	request.BenchChan = c
	return request
}

// WithContentType with contenttype
func (request *Request) WithContentType(ct ContentType) *Request {
	request.ContentType = ct
	return request
}

// WithHeader add header to request
func (request *Request) WithHeader(name string, value string) *Request {
	if request.Headers == nil {
		request.Headers = make(map[string]string)
	}

	request.Headers[name] = value
	return request
}

// BuildClient return client
func (request *Request) BuildClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: request.MaxConnectionsPerHost,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: request.Config.IgnoreCert,
			},
			MaxIdleConns:        5,
			MaxIdleConnsPerHost: 5,
			DisableKeepAlives:   true,
		},
		Timeout: 0,
	}
}

// DoHTTPRequest do plain http request
func (request *Request) DoHTTPRequest() (*http.Response, error) {
	client := request.BuildClient()

	// Build url
	u, err := url.Parse(request.Config.URL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, string(request.Endpoint))

	var reader io.Reader

	// Use correct payload
	if request.RequestType == JSONRequestType {
		// Encode data
		var err error
		bytePayload, err := json.Marshal(request.Payload)
		if err != nil {
			return nil, err
		}

		reader = bytes.NewReader(bytePayload)
	} else if request.RequestType == RawRequestType {
		switch request.Payload.(type) {
		case []byte:
			reader = bytes.NewReader((request.Payload).([]byte))
		case io.Reader:
			reader = (request.Payload).(io.Reader)
		case io.PipeReader:
			reader = (request.Payload).(*io.PipeReader)
		}
	}

	if reader == nil {
		reader = bytes.NewBuffer([]byte(""))
	}

	// Bulid request
	req, _ := http.NewRequest(string(request.Method), u.String(), reader)

	// Set contenttype header
	req.Header.Set("Content-Type", string(request.ContentType))

	if request.Compressed {
		req.Header.Set("Content-Encoding", string("gzip"))
	}

	for headerKey, headerValue := range request.Headers {
		req.Header.Set(headerKey, headerValue)
	}

	// Set Authorization header
	if request.Authorization != nil {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", string(request.Authorization.Type), request.Authorization.Palyoad))
	}

	return client.Do(req)
}

// Do a better request method
func (request Request) Do(retVar interface{}) (*RestRequestResponse, error) {
	resp, err := request.DoHTTPRequest()
	if err != nil || resp == nil {
		return nil, err
	}

	defer resp.Body.Close()

	var response *RestRequestResponse

	response = &RestRequestResponse{
		HTTPCode: resp.StatusCode,
		Headers:  &resp.Header,
	}

	if resp.StatusCode == 200 {
		response.Status = ResponseSuccess
	} else {
		response.Status = ResponseError
	}

	response.Message = ""

	// Only fill retVar if response was successful
	if response.Status == ResponseSuccess && retVar != nil {
		// Read response
		d, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// Parse response into retVar
		err = json.Unmarshal(d, &retVar)
		if err != nil {
			return nil, err
		}
	} else if response.Status == ResponseError {
		var errRes ErrorResponse

		// Read response
		d, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// Parse response into retVar
		err = json.Unmarshal(d, &errRes)
		if err != nil {
			return nil, err
		}

		response.Message = fmt.Sprintf("%s", errRes.Message)
	}

	return response, nil
}

package libdatamanager

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidResponseHeaders error on missing or malformed response headers
	ErrInvalidResponseHeaders = errors.New("Invalid response headers")
	// ErrInvalidAuthorizationMethod error if authorization method is not implemented
	ErrInvalidAuthorizationMethod = errors.New("Invalid request authorization method")
	// ErrNonOkResponsecode if requests responsecode is non http.ok
	ErrNonOkResponsecode = errors.New("non ok response")
)

// ResponseErr response error
type ResponseErr struct {
	HTTPStatusCode   int
	ResopnseMessage  string
	ReceivedResponse bool
	Err              error
}

func (reserr *ResponseErr) Error() string {
	return fmt.Sprintf("HTTPCode: %d; Status: %s", reserr.HTTPStatusCode, reserr.ResopnseMessage)
}

// NewErrorFromResponse return error from response
func NewErrorFromResponse(r *RestRequestResponse, err ...error) *ResponseErr {
	var e error

	if len(err) > 0 && err[0] != nil {
		e = err[0]
	}

	if r != nil && r.HTTPCode != 0 {
		return &ResponseErr{
			ReceivedResponse: true,
			HTTPStatusCode:   r.HTTPCode,
			ResopnseMessage:  r.Message,
			Err:              e,
		}
	}

	return &ResponseErr{
		Err: e,
	}
}

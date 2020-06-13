package libdatamanager

// LibDM data required in all requests
type LibDM struct {
	Config                *RequestConfig
	MaxConnectionsPerHost int
}

// NewLibDM create new libDM "class"
func NewLibDM(config *RequestConfig) *LibDM {
	return &LibDM{
		Config: config,
	}
}

// WithMaxConnections per host
func (libdm *LibDM) WithMaxConnections(maxConnecetions int) *LibDM {
	libdm.MaxConnectionsPerHost = maxConnecetions
	return libdm
}

// Request do a request using libdm
func (libdm LibDM) Request(ep Endpoint, payload, response interface{}, authorized bool) (*RestRequestResponse, error) {
	req := libdm.NewRequest(ep, payload)
	req.WithConnectionLimit(libdm.MaxConnectionsPerHost)
	if authorized {
		req.WithAuthFromConfig()
	}
	resp, err := req.Do(response)

	if err != nil || resp.Status == ResponseError {
		return nil, NewErrorFromResponse(resp, err)
	}

	return resp, nil
}

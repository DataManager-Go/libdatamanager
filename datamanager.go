package libdatamanager

// LibDM data required in all requests
type LibDM struct {
	Config *RequestConfig
}

// NewLibDM create new libDM "class"
func NewLibDM(config *RequestConfig) *LibDM {
	return &LibDM{
		Config: config,
	}
}

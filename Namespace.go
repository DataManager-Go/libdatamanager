package libdatamanager

// NamespaceType type of namespace
type NamespaceType uint8

// Namespace types
const (
	UserNamespaceType NamespaceType = iota
	CustomNamespaceType
)

func (libdm LibDM) namespaceRequest(action uint8, name, newName string, customNs bool) (*StringResponse, error) {
	var response StringResponse
	endpoint := namespaceActionToEndpoint(action)

	nsType := UserNamespaceType
	if customNs {
		nsType = CustomNamespaceType
	}

	resp, err := NewRequest(endpoint, NamespaceRequest{
		Namespace: name,
		NewName:   newName,
		Type:      nsType,
	}, libdm.Config).WithAuthFromConfig().Do(&response)

	if err != nil {
		return nil, NewErrorFromResponse(resp)
	}

	if resp.Status == ResponseError {
		return nil, ErrResponseError
	}

	return &response, nil
}

// CreateNamespace creates a namespace
func (libdm LibDM) CreateNamespace(name string, customNS bool) (*StringResponse, error) {
	return libdm.namespaceRequest(1, name, "", customNS)
}

// UpdateNamespace update a namespace
func (libdm LibDM) UpdateNamespace(name, newName string, customNS bool) (*StringResponse, error) {
	return libdm.namespaceRequest(2, name, newName, customNS)
}

// DeleteNamespace update a namespace
func (libdm LibDM) DeleteNamespace(name string) (*StringResponse, error) {
	return libdm.namespaceRequest(0, name, "", false)
}

func namespaceActionToEndpoint(action uint8) (endpoint Endpoint) {
	switch action {
	case 0:
		endpoint = EPNamespaceDelete
	case 1:
		endpoint = EPNamespaceCreate
	case 2:
		endpoint = EPNamespaceUpdate
	}

	return
}

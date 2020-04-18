package libdatamanager

func (libdm LibDM) namespaceRequest(action uint8, name, newName string) (*StringResponse, error) {
	var response StringResponse
	endpoint := namespaceActionToEndpoint(action)

	// Do http request
	if _, err := libdm.Request(endpoint, &NamespaceRequest{
		Namespace: name,
		NewName:   newName,
	}, &response, true); err != nil {
		return nil, err
	}

	return &response, nil
}

// CreateNamespace creates a namespace
func (libdm LibDM) CreateNamespace(name string) (*StringResponse, error) {
	return libdm.namespaceRequest(1, name, "")
}

// UpdateNamespace update a namespace
func (libdm LibDM) UpdateNamespace(name, newName string) (*StringResponse, error) {
	return libdm.namespaceRequest(2, name, newName)
}

// DeleteNamespace update a namespace
func (libdm LibDM) DeleteNamespace(name string) (*StringResponse, error) {
	return libdm.namespaceRequest(0, name, "")
}

// GetNamespaces get all namespaces
func (libdm LibDM) GetNamespaces() (*StringSliceResponse, error) {
	var resp StringSliceResponse

	// Do http request
	if _, err := libdm.Request(EPNamespaceList, nil, &resp, true); err != nil {
		return nil, err
	}

	return &resp, nil
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

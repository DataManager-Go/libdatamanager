package libdatamanager

// Attribute attribute for file (tag/group)
type Attribute string

// Attributes
const (
	TagAttribute   Attribute = "tag"
	GroupAttribute Attribute = "group"
)

// Do an attribute request (update/delete group or tag). action: 0 - delete, 1 - update, 2 - get, 3 - create
func (libdm LibDM) attributeRequest(attribute Attribute, action uint8, namespace string, name string, response interface{}, newName ...string) (*RestRequestResponse, error) {
	var endpoint Endpoint

	// Pick right endpoint
	switch action {
	case 0:
		if attribute == GroupAttribute {
			endpoint = EPGroupDelete
		} else {
			endpoint = EPTagDelete
		}
	case 1:
		if attribute == GroupAttribute {
			endpoint = EPGroupUpdate
		} else {
			endpoint = EPTagUpdate
		}
	case 2:
		if attribute == GroupAttribute {
			endpoint = EPGroups
		} else {
			endpoint = EPTags
		}
	case 3:
		if attribute == GroupAttribute {
			endpoint = EPGroupCreate
		} else {
			endpoint = EPTagCreate
		}
	}

	// Build request
	request := UpdateAttributeRequest{
		Name:      name,
		Namespace: namespace,
	}

	// Set NewName on update request
	if action == 1 {
		request.NewName = newName[0]
	}

	var resp *RestRequestResponse
	var err error

	// Do http request
	if resp, err = libdm.Request(endpoint, &request, response, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// CreateAttribute update an attribute
func (libdm LibDM) CreateAttribute(attribute Attribute, namespace, name string) (*RestRequestResponse, error) {
	return libdm.attributeRequest(attribute, 3, namespace, name, nil)
}

// UpdateAttribute update an attribute
func (libdm LibDM) UpdateAttribute(attribute Attribute, namespace, name, newName string) (*RestRequestResponse, error) {
	return libdm.attributeRequest(attribute, 1, namespace, name, nil, newName)
}

// DeleteAttribute update an attribute
func (libdm LibDM) DeleteAttribute(attribute Attribute, namespace, name string) (*RestRequestResponse, error) {
	return libdm.attributeRequest(attribute, 0, namespace, name, nil)
}

// GetTags returns an array of attributes containing tags available in given namespace
func (libdm LibDM) GetTags(namespace string) ([]Attribute, error) {
	var attributes []Attribute
	_, err := libdm.attributeRequest(TagAttribute, 2, namespace, "", &attributes)
	if err != nil {
		return nil, err
	}

	return attributes, nil
}

// GetGroups returns an array of attributes containing groups available in given namespace
func (libdm LibDM) GetGroups(namespace string) ([]Attribute, error) {
	var attributes []Attribute
	_, err := libdm.attributeRequest(GroupAttribute, 2, namespace, "", &attributes)
	if err != nil {
		return nil, err
	}

	return attributes, nil
}

// GetUserAttributeData get attribute data for an user
func (libdm LibDM) GetUserAttributeData() (*UserAttributeDataResponse, error) {
	var response *UserAttributeDataResponse

	_, err := libdm.Request(EPAttributes, nil, &response, true)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// SortByName sorts NamespaceInfo by name
type SortByName []Namespaceinfo

func (a SortByName) Len() int           { return len(a) }
func (a SortByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

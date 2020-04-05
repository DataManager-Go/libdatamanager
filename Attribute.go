package libdatamanager

import (
	"fmt"
)

// Attribute attribute for file (tag/group)
type Attribute string

// Attributes
const (
	TagAttribute   Attribute = "tag"
	GroupAttribute Attribute = "group"
)

// Do an attribute request (update/delete group or tag). action: 0 - delete, 1 - update
func (libdm LibDM) attributeRequest(attribute Attribute, namespace string, action uint8, name string, newName ...string) (*RestRequestResponse, error) {
	var endpoint Endpoint

	// Pick right endpoint
	if action == 1 {
		if attribute == GroupAttribute {
			endpoint = EPGroupUpdate
		} else {
			endpoint = EPTagUpdate
		}
	} else if action == 0 {
		if attribute == GroupAttribute {
			endpoint = EPGroupDelete
		} else {
			endpoint = EPTagDelete
		}
	}

	// Build request
	request := UpdateAttributeRequest{
		Name:      name,
		Namespace: namespace,
	}

	// Add new name on update request
	if action == 1 {
		request.NewName = newName[0]
	}

	resp, err := NewRequest(endpoint, request, libdm.Config).WithAuth(Authorization{
		Type:    Bearer,
		Palyoad: libdm.Config.SessionToken,
	}).Do(nil)

	if err != nil {
		if resp != nil {
			fmt.Println("http:", resp.HTTPCode)
			return nil, NewErrorFromResponse(resp)
		}
	}

	if resp.Status == ResponseError {
		return resp, ErrResponseError
	}

	return resp, nil
}

// UpdateAttribute update an attribute
func (libdm LibDM) UpdateAttribute(attribute Attribute, namespace, name, newName string) (*RestRequestResponse, error) {
	return libdm.attributeRequest(attribute, namespace, 1, name, newName)
}

// DeleteAttribute update an attribute
func (libdm LibDM) DeleteAttribute(attribute Attribute, namespace, name string) (*RestRequestResponse, error) {
	return libdm.attributeRequest(attribute, namespace, 0, name)
}

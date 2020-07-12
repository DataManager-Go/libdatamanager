package libdatamanager

import "strings"

// Login login into the server
func (libdm LibDM) Login(username, password string) (*LoginResponse, error) {
	var response LoginResponse

	// Do http request
	resp, err := libdm.NewRequest(EPLogin, CredentialsRequest{
		Password:  password,
		Username:  strings.ToLower(username),
		MachineID: libdm.Config.MachineID,
	}).Do(&response)

	// Return new error on ... error
	if err != nil || resp.Status == ResponseError {
		return nil, NewErrorFromResponse(resp, err)
	}

	return &response, nil
}

// Register create a new account. Return true on success
func (libdm LibDM) Register(username, password string) (*RestRequestResponse, error) {
	// Do http request
	resp, err := libdm.NewRequest(EPRegister, CredentialsRequest{
		Username: strings.ToLower(username),
		Password: password,
	}).Do(nil)

	if err != nil {
		return resp, NewErrorFromResponse(resp, err)
	}

	return resp, nil
}

// Stats for user
func (libdm LibDM) Stats(namespace string) (*StatsResponse, error) {
	var response StatsResponse

	if _, err := libdm.Request(EPUserStats, &StatsRequestStruct{
		Namespace: namespace,
	}, &response, true); err != nil {
		return nil, err
	}

	return &response, nil
}

// Ping pings a server the REST way to
// ensure it is reachable
func (libdm LibDM) Ping() (*StringResponse, error) {
	var response StringResponse

	// Do ping request
	req := libdm.NewRequest(EPPing, PingRequest{Payload: "ping"})
	if libdm.Config.SessionToken != "" {
		req.WithAuthFromConfig()
	}
	_, err := req.Do(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

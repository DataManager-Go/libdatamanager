package libdatamanager

// Login login into the server
func (libdm LibDM) Login(username, password string) (*LoginResponse, error) {
	var response LoginResponse

	// Do http request
	resp, err := NewRequest(EPLogin, CredentialsRequest{
		Password:  password,
		Username:  username,
		MachineID: libdm.Config.MachineID,
	}, libdm.Config).Do(&response)

	// Return new error on ... error
	if err != nil || resp.Status == ResponseError {
		return nil, NewErrorFromResponse(resp, err)
	}

	return &response, nil
}

// Register create a new account. Return true on success
func (libdm LibDM) Register(username, password string) (*RestRequestResponse, error) {
	// Do http request
	resp, err := NewRequest(EPRegister, CredentialsRequest{
		Username: username,
		Password: password,
	}, libdm.Config).Do(nil)

	if err != nil {
		return resp, NewErrorFromResponse(resp, err)
	}

	return resp, nil
}

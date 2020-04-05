package libdatamanager

// Login login into the server
func (libdm LibDM) Login(username, password string) (*LoginResponse, error) {
	var response LoginResponse

	// Do request
	resp, err := NewRequest(EPLogin, CredentialsRequest{
		Password:  password,
		Username:  username,
		MachineID: libdm.Config.MachineID,
	}, libdm.Config).Do(&response)

	if err != nil {
		return nil, NewErrorFromResponse(resp, err)
	}

	if resp.Status == ResponseError {
		return nil, NewErrorFromResponse(resp)
	} else if resp.Status == ResponseSuccess {
		return &response, nil
	}

	return nil, nil
}

// Register create a new account. Return true on success
func (libdm LibDM) Register(username, password string) (*RestRequestResponse, error) {
	// Do request
	resp, err := NewRequest(EPRegister, CredentialsRequest{
		Username: username,
		Password: password,
	}, libdm.Config).Do(nil)

	if err != nil {
		return resp, NewErrorFromResponse(resp, err)
	}

	return resp, nil
}

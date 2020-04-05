package libdatamanager

// Login login into the server
func Login(confg *RequestConfig, username, password string) (*LoginResponse, error) {
	var response LoginResponse

	// Do request
	resp, err := NewRequest(EPLogin, CredentialsRequest{
		Password:  password,
		Username:  username,
		MachineID: confg.MachineID,
	}, confg).Do(&response)

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

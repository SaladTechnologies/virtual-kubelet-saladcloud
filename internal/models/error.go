package models

type InvalidClientSecretError struct{}

func NewInvalidClientSecretError() *InvalidClientSecretError {
	return &InvalidClientSecretError{}
}

func (e *InvalidClientSecretError) Error() string {
	return "invalid SaladCloud client api key"
}

//Resolve error from response

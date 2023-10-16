package models

import (
	"fmt"
	"net/http"
)

type APIError struct {
	StatusCode int
	Message    string
}

func (a *APIError) Error() string {
	message := fmt.Sprintf(
		"a %d error was returned from SaladCloud: \"%s\"",
		a.StatusCode,
		a.Message,
	)

	return message
}

func NewSaladCloudError(err error, response *http.Response) error {
	if response == nil {
		return err
	}
	return &APIError{
		StatusCode: response.StatusCode,
		Message:    err.Error(),
	}
}

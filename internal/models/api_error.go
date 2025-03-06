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
	// If there is no response or the status code indicates success, return the original error.
	if response == nil || response.StatusCode < 400 {
		return err
	}

	// Use an empty message if err is nil; otherwise, use err.Error()
	message := ""
	if err != nil {
		message = err.Error()
	}

	return &APIError{
		StatusCode: response.StatusCode,
		Message:    message,
	}
}

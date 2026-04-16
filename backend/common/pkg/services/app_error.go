package services

import "fmt"

type AppInternalError struct {
	Message string
}

func (e *AppInternalError) Error() string {
	return e.Message
}

func NewAppInternalError(format string, a ...any) error {
	msg := fmt.Sprintf(format, a...)
	return &AppInternalError{Message: msg}
}

type AppValidationError struct {
	Form FormValidation
}

func (e *AppValidationError) Error() string {
	return e.Form.Message
}
func NewAppValidationError(message string, path string) error {
	return &AppValidationError{Form: FormValidation{
		Message: message, Path: path,
	}}
}

// filepath: internal/services/service_errors.go
package services

import "errors"

// Standard errors returned by the service layer.
var (
	ErrNotFound     = errors.New("not found")
	ErrValidation   = errors.New("validation failed")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrUnsupported  = errors.New("unsupported media type")
	ErrDependencies = errors.New("dependency check failed")
)

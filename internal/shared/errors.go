package shared

type Error string

// Implement the error interface
func (e Error) Error() string { return string(e) }

//------------
// Definitions
//------------

// cli errors
const (
	ErrorCreateFile = Error("could not create the file")
	ErrorEncodeFile = Error("could not encode to file")
)

// repository errors
const ErrUserNotFound = Error("user not found")
const ErrInvalidName = Error("invalid name")

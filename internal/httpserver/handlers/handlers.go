package handlers

// container holding all other "subhandlers"
// each subhandler contains all required data and types to respond
// to HTTP calls
type Handlers struct {
	InfoHandler InfoHandler
}

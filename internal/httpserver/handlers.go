package httpserver

import (
	ah "mediahub_oss/internal/httpserver/audithandler"
	dbh "mediahub_oss/internal/httpserver/databasehandler"
	eh "mediahub_oss/internal/httpserver/entryhandler"
	ih "mediahub_oss/internal/httpserver/infohandler"
	th "mediahub_oss/internal/httpserver/tokenhandler"
	uh "mediahub_oss/internal/httpserver/userhandler"
)

// container holding all other "subhandlers"
// each subhandler contains all required data and types to respond
// to HTTP calls
type Handlers struct {
	// Handlers
	InfoHandler     ih.InfoHandler
	EntryHandler    eh.EntryHandler
	DatabaseHandler dbh.DatabaseHandler
	UserHandler     uh.UserHandler
	TokenHandler    th.TokenHandler
	AuditHandler    ah.AuditHandler
}

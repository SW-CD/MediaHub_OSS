// filepath: cmd/mediahub/main.go
package main

import (
	"embed"
	"mediahub/internal/cli"

	// Import docs for Swagger
	_ "mediahub/docs"
)

//go:embed all:frontend_embed/browser
var frontendFS embed.FS

// @title SWCD MediaHub-API
// @version 1.2.0
// @description This is a sample server for a file store.
// @contact.name Christian Dengler
// @contact.url https://www.swcd.lu
// @contact.email denglerchr@gmail.com
// @BasePath /api
// @schemes http
// @securityDefinitions.basic BasicAuth
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and a JWT token.
// @import encoding/json

func main() {
	// Delegate all execution to the CLI package
	cli.Execute(frontendFS)
}

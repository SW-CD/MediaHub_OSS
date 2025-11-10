// internal/web/web.go
// Package web handles serving the embedded frontend application.
package web

import (
	"bytes"
	"embed"
	"io" // <-- Import io
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// spaHandler serves a single-page application from an embedded filesystem.
type spaHandler struct {
	contentFS fs.FS  // The embedded filesystem (stripped of prefix)
	indexPath string // e.g., "index.html"
}

// ServeHTTP handles serving the SPA.
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the path from the request.
	reqPath := r.URL.Path

	// Clean the path, remove leading slash for fs.Open
	if strings.HasPrefix(reqPath, "/") {
		reqPath = reqPath[1:]
	}

	// Use 'path.Clean' for FS paths, not 'filepath.Clean'
	filePath := path.Clean(reqPath)

	// Use path as the file name, or indexPath if path is empty.
	if filePath == "" || filePath == "." {
		filePath = h.indexPath
	}

	// Try to open the requested file (e.g., "assets/logo.png")
	file, err := h.contentFS.Open(filePath)
	if err != nil {
		// If it doesn't exist, serve the index.html
		if os.IsNotExist(err) {
			indexBytes, err := fs.ReadFile(h.contentFS, h.indexPath)
			if err != nil {
				http.Error(w, "Internal server error: index.html not found", http.StatusInternalServerError)
				log.Printf("ERROR: spaHandler could not find %s in embedded FS: %v", h.indexPath, err)
				return
			}
			reader := bytes.NewReader(indexBytes)
			http.ServeContent(w, r, h.indexPath, time.Time{}, reader)
			return
		}

		// Another error (e.g., permission)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("ERROR: spaHandler error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Get FileInfo for http.ServeContent
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("ERROR: spaHandler error stating file %s: %v", filePath, err)
		return
	}

	// --- PERFORMANCE FIX ---
	// We must type-assert the fs.File to io.ReadSeeker for http.ServeContent.
	// The file from embed.FS *does* implement this, but the fs.File interface
	// does not guarantee it, so the compiler fails without this check.
	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		// This should not happen with embed.FS, but we handle it gracefully
		// by falling back to reading the file into memory.
		log.Printf("WARN: spaHandler file %s does not implement io.ReadSeeker. Falling back to memory buffer.", filePath)
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Printf("ERROR: spaHandler error reading file %s: %v", filePath, err)
			return
		}
		reader := bytes.NewReader(fileBytes)
		http.ServeContent(w, r, filePath, fileInfo.ModTime(), reader)
		return
	}

	// File exists and implements io.ReadSeeker, serve it directly
	http.ServeContent(w, r, filePath, fileInfo.ModTime(), seeker)
}

// AddRoutes mounts the frontend handler to the main router.
// It now takes an embed.FS
func AddRoutes(router *mux.Router, content embed.FS, indexPath string) {
	subFS, err := fs.Sub(content, "frontend_embed/browser")
	if err != nil {
		log.Fatalf("Failed to create sub FS for frontend: %v", err)
	}

	spa := spaHandler{
		contentFS: subFS,
		indexPath: indexPath,
	}
	router.PathPrefix("/").Handler(spa)
}

package ffmpeg

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// LocalStreamServer acts as an internal loopback bridge, allowing FFmpeg to securely
// access in-memory data streams via HTTP Range requests.
type LocalStreamServer struct {
	server   *http.Server
	listener net.Listener
	sessions map[string]*StreamSession
	mu       sync.RWMutex // Protects the sessions map from concurrent map writes/reads
	logger   *slog.Logger
	baseURL  string // The dynamic base URL of the local server
	cancel   context.CancelFunc
}

type StreamSession struct {
	id        string
	readerAt  io.ReaderAt // Replaces io.ReadSeeker to allow stateless concurrent reads
	size      int64       // Required for io.NewSectionReader
	token     string
	expiresAt time.Time
}

// NewLocalStreamServer initializes and starts the internal server.
func NewLocalStreamServer(logger *slog.Logger) (*LocalStreamServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind local stream server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	ls := &LocalStreamServer{
		listener: listener,
		sessions: make(map[string]*StreamSession),
		logger:   logger,
		baseURL:  fmt.Sprintf("http://%s", listener.Addr().String()),
		cancel:   cancel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /stream/{id}", ls.handleStream)

	ls.server = &http.Server{Handler: mux}

	go func() {
		if err := ls.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("Local stream server crashed", "error", err)
		}
	}()

	go ls.startSweeper(ctx)

	logger.Info("Internal loopback server started", "url", ls.baseURL)
	return ls, nil
}

// Shutdown gracefully closes the internal server.
func (l *LocalStreamServer) Shutdown(ctx context.Context) error {
	l.cancel()
	return l.server.Shutdown(ctx)
}

// Register securely mounts an in-memory stream and returns its unique ID and full FFmpeg-ready URL.
func (l *LocalStreamServer) Register(stream io.ReadSeeker, ttl time.Duration) (string, string, error) {
	id := generateRandomHex(16)
	token := generateRandomHex(16)

	// Determine the total size of the stream
	size, _ := stream.Seek(0, io.SeekEnd)
	stream.Seek(0, io.SeekStart) // Rewind

	// Ensure the stream supports concurrent stateless reads (io.ReaderAt)
	readerAt, ok := stream.(io.ReaderAt)
	if !ok {
		// Fallback: If it's a weird stream type, read it into memory so it becomes a bytes.Reader
		l.logger.Warn("Stream does not implement io.ReaderAt, buffering into RAM")
		data, err := io.ReadAll(stream)
		if err != nil {
			return "", "", fmt.Errorf("failed to buffer stream into memory: %w", err) // <-- Handle error safely!
		}
		readerAt = bytes.NewReader(data)
		size = int64(len(data))
	}

	session := &StreamSession{
		id:        id,
		readerAt:  readerAt,
		size:      size,
		token:     token,
		expiresAt: time.Now().Add(ttl),
	}

	l.mu.Lock()
	l.sessions[id] = session
	l.mu.Unlock()

	fullURL := fmt.Sprintf("%s/stream/%s?token=%s", l.baseURL, id, token)
	return id, fullURL, nil
}

func (l *LocalStreamServer) Unregister(id string) {
	l.mu.Lock()
	delete(l.sessions, id)
	l.mu.Unlock()
}

// handleStream is the internal endpoint that FFmpeg requests data from.
func (l *LocalStreamServer) handleStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	token := r.URL.Query().Get("token")

	l.mu.RLock()
	session, exists := l.sessions[id]
	l.mu.RUnlock()

	if !exists || session.token != token {
		http.Error(w, "Unauthorized or Stream Not Found", http.StatusUnauthorized)
		return
	}

	if time.Now().After(session.expiresAt) {
		http.Error(w, "Stream Session Expired", http.StatusGone)
		return
	}

	// Give THIS specific HTTP request its own independent cursor!
	// This prevents the race condition when FFmpeg opens multiple concurrent threads.
	independentStream := io.NewSectionReader(session.readerAt, 0, session.size)

	http.ServeContent(w, r, "stream.bin", time.Time{}, independentStream)
}

func (l *LocalStreamServer) startSweeper(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.mu.Lock()
			now := time.Now()
			for id, session := range l.sessions {
				if now.After(session.expiresAt) {
					delete(l.sessions, id)
					l.logger.Debug("Sweeper removed expired stream session", "id", id)
				}
			}
			l.mu.Unlock()
		}
	}
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

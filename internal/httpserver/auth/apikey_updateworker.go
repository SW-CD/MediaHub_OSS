package auth

import (
	"context"
	"log"
	"mediahub_oss/internal/repository"
	"time"
)

// apiKeyUpdateWorker runs in the background, debouncing updates and flushing them every 5 seconds.
func (am *AuthMiddleware) apiKeyUpdateWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Map to deduplicate updates (debouncing)
	pendingUpdates := make(map[repository.ULID]struct{})

	for {
		select {
		case keyID := <-am.apiKeyUpdateChan:
			pendingUpdates[keyID] = struct{}{} // Deduplicate
		case <-ticker.C:
			if len(pendingUpdates) == 0 {
				continue
			}

			for keyID := range pendingUpdates {
				if err := am.Repo.UpdateAPIKeyLastUsed(context.Background(), keyID, 0); err != nil {
					log.Printf("Failed to update last_used_at for api_key %s: %v", keyID, err)
				}
			}

			// Fast map clear (Go 1.21+ built-in)
			clear(pendingUpdates)
		}
	}
}

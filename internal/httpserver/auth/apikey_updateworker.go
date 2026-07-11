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

	// Map to deduplicate updates (debouncing), storing the oldest unused timestamp
	pendingUpdates := make(map[repository.ULID]time.Time)

	for {
		select {
		case req := <-am.apiKeyUpdateChan:
			pendingUpdates[req.KeyID] = req.UsedAt // Store the latest timestamp

		case <-ticker.C:
			if len(pendingUpdates) == 0 {
				continue
			}

			for keyID, usedAt := range pendingUpdates {
				duration := time.Since(usedAt)
				if err := am.Repo.UpdateAPIKeyLastUsed(context.Background(), keyID, duration); err != nil {
					log.Printf("Failed to update last_used_at for api_key %s: %v", keyID, err)
				}
			}

			// Fast map clear (Go 1.21+ built-in)
			clear(pendingUpdates)
		}
	}
}

package auth

import (
	"context"
	"log"
	"mediahub_oss/internal/repository"
	"time"
)

// apiKeyUpdateWorker runs in the background, debouncing updates and flushing them every 5 seconds or upon shutdown.
func (am *AuthMiddleware) apiKeyUpdateWorker() {
	defer close(am.doneChan)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Map to deduplicate updates (debouncing), storing the oldest unused timestamp
	pendingUpdates := make(map[repository.ULID]time.Time)

	flush := func() {
		if len(pendingUpdates) == 0 || am.Repo == nil {
			return
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

	for {
		select {
		case <-am.stopChan:
			// Drain any remaining buffered requests in apiKeyUpdateChan
			for {
				select {
				case req := <-am.apiKeyUpdateChan:
					pendingUpdates[req.KeyID] = req.UsedAt
				default:
					goto drained
				}
			}
		drained:
			flush()
			return

		case req := <-am.apiKeyUpdateChan:
			pendingUpdates[req.KeyID] = req.UsedAt // Store the latest timestamp

		case <-ticker.C:
			flush()
		}
	}
}

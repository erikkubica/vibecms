package uploads

import (
	"context"
	"log"
	"time"
)

// StartCleanupLoop runs a background ticker that sweeps expired-not-finalized
// rows + their temp files every interval. Cancellable via ctx — use
// bgCancel() at shutdown to stop the loop.
func StartCleanupLoop(ctx context.Context, store *Store, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := store.Cleanup(); err != nil {
					log.Printf("uploads: cleanup tick failed: %v", err)
				} else if n > 0 {
					log.Printf("uploads: swept %d expired uploads", n)
				}
			}
		}
	}()
}

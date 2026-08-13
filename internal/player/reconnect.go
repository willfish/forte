package player

import (
	"time"

	mpv "github.com/gen2brain/go-mpv"
)

const radioReconnectMaxDelay = 30 * time.Second

// StreamEndIsError reports whether an mpv end-file reason should be treated as
// a lost stream rather than a finished playlist item.
func StreamEndIsError(reason mpv.Reason, reconnectOnEOF bool) bool {
	switch reason {
	case mpv.EndFileError:
		return true
	case mpv.EndFileEOF:
		return reconnectOnEOF
	default:
		return false
	}
}

// RadioReconnectDelay returns a full-jitter backoff for a 1-based attempt.
// jitter is in [0, 1]; values outside that range are clamped.
func RadioReconnectDelay(attempt int, jitter float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	exp := attempt - 1
	if exp > 5 {
		exp = 5
	}
	max := time.Second << exp
	if max > radioReconnectMaxDelay {
		max = radioReconnectMaxDelay
	}
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	return time.Duration(float64(max) * jitter)
}

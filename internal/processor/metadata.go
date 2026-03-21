package processor

import (
	"math"
	"strings"

	"github.com/konstfish/pumice/internal/types"
)

// PageMetadata is an alias for the shared type.
type PageMetadata = types.PageMetadata

// estimateReadingTime returns estimated minutes to read the given text.
// Uses 238 words per minute (average adult reading speed).
func estimateReadingTime(text string) int {
	words := len(strings.Fields(text))
	minutes := float64(words) / 238.0
	if minutes < 1 && words > 0 {
		return 1
	}
	return int(math.Ceil(minutes))
}

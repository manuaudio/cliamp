package subsonicapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/bjarneo/cliamp/playlist"
	"github.com/bjarneo/cliamp/provider"
)

var _ provider.TrackRater = (*Client)(nil)

// CanRateTrack reports whether track came from this Subsonic server and the
// dialect supports server-side numeric ratings. It performs no I/O.
func (c *Client) CanRateTrack(track playlist.Track) bool {
	return c.d.ratingSupported() && track.Meta(c.d.metaKey()) != ""
}

// SetTrackRating sets track's rating on the server. A rating of 0 removes it.
func (c *Client) SetTrackRating(ctx context.Context, track playlist.Track, rating int) error {
	if !c.d.ratingSupported() {
		return fmt.Errorf("%s: ratings are not supported", c.prefix())
	}
	if rating < 0 || rating > 5 {
		return fmt.Errorf("%s: rating must be between 0 and 5, got %d", c.prefix(), rating)
	}
	id := track.Meta(c.d.metaKey())
	if id == "" {
		return fmt.Errorf("%s: %q has no song ID", c.prefix(), track.Path)
	}
	var result struct{}
	return c.subsonicGet(ctx, "setRating", url.Values{
		"id":     {id},
		"rating": {strconv.Itoa(rating)},
	}, &result)
}

package model

import (
	"context"
	"maps"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bjarneo/cliamp/favorites"
	"github.com/bjarneo/cliamp/playlist"
	"github.com/bjarneo/cliamp/provider"
)

const trackRatingTimeout = 30 * time.Second

type trackRatingSyncedMsg struct {
	provider string
	track    playlist.Track
	cursor   int
	request  uint64
	rating   int
	err      error
}

func sameRatingTrack(a, b playlist.Track) bool {
	return a.Path == b.Path && maps.Equal(a.ProviderMeta, b.ProviderMeta)
}

func (m *Model) findTrackRater(track playlist.Track) (provider.TrackRater, string) {
	return findCapable(m, func(r provider.TrackRater) bool { return r.CanRateTrack(track) })
}

func (m *Model) rateSelectedTrack(rating int) tea.Cmd {
	if rating < 0 || rating > 5 || m.playlist == nil || m.plCursor < 0 || m.plCursor >= m.playlist.Len() {
		return nil
	}
	track, ok := m.playlist.Track(m.plCursor)
	if !ok {
		return nil
	}
	rater, name := m.findTrackRater(track)
	if rater == nil {
		m.status.Warningf(statusTTLMedium, "Rating is not available for %s", track.DisplayName())
		return nil
	}
	m.ratingRequest++
	request := m.ratingRequest
	cursor := m.plCursor
	if m.ratingSync == nil {
		m.ratingSync = &favorites.SyncQueue{}
	}
	songID := track.Meta(provider.MetaNavidromeID)
	if songID == "" {
		songID = track.Path
	}
	syncKey := name + "\x00" + songID
	run := m.ratingSync.Enqueue(syncKey, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), trackRatingTimeout)
		defer cancel()
		return rater.SetTrackRating(ctx, track, rating)
	})
	return func() tea.Msg {
		ran, err := run()
		if !ran {
			return nil
		}
		return trackRatingSyncedMsg{
			provider: name,
			track:    track,
			cursor:   cursor,
			request:  request,
			rating:   rating,
			err:      err,
		}
	}
}

func (m *Model) handleTrackRatingSynced(msg trackRatingSyncedMsg) {
	if msg.request != m.ratingRequest || m.playlist == nil || msg.cursor != m.plCursor {
		return
	}
	track, ok := m.playlist.Track(msg.cursor)
	if !ok || !sameRatingTrack(track, msg.track) {
		return
	}
	if msg.err != nil {
		m.status.Warningf(statusTTLMedium, "%s did not save rating: %s", msg.provider, msg.err)
		return
	}
	track.UserRating = msg.rating
	m.playlist.SetTrack(msg.cursor, track)
	if m.playingTrackActive && !m.playbackDetached && m.playlist.Index() == msg.cursor && sameRatingTrack(m.playingTrack, msg.track) {
		m.playingTrack.UserRating = msg.rating
	}
	if msg.rating == 0 {
		m.status.Showf(statusTTLMedium, "Cleared rating: %s", track.DisplayName())
	} else {
		m.status.Showf(statusTTLMedium, "Rated %d/5: %s", msg.rating, track.DisplayName())
	}
}

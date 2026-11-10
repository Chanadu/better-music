package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

func syncCall(t *testing.T, h *Handler, kind, op, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, e := json.Marshal(body)
	if e != nil {
		t.Fatal(e)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sync/{kind}/pull", h.PullLibrary)
	mux.HandleFunc("POST /api/sync/{kind}/push", h.PushLibrary)
	r := httptest.NewRequest("POST", "/api/sync/"+kind+"/"+op, bytes.NewReader(raw))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	middleware.Auth(mux, h.Config.JWTSecret).ServeHTTP(w, r)
	return w
}

func initialPull(t *testing.T, h *Handler, kind, token string) LibraryCheckpoint {
	w := syncCall(t, h, kind, "pull", token, PullLibraryRequest{Limit: 100})
	status(t, w, 200)

	var page struct {
		Checkpoint LibraryCheckpoint `json:"checkpoint"`
	}

	if e := json.Unmarshal(w.Body.Bytes(), &page); e != nil {
		t.Fatal(e)
	}

	return page.Checkpoint
}

func artistDocument(id, name, edited string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"id":%q,"name":%q,"cover_url":null,"spotify_id":null,"created_at":"2026-01-01T00:00:00.000Z","edited_at":%q,"_deleted":false}`, id, name, edited))
}

func albumDocument(id, artist, title, edited string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"id":%q,"artist_id":%q,"title":%q,"cover_url":null,"spotify_id":null,"year":2024,"listened":true,"rating":8,"comment":"Keep these notes","listened_at":"2026-01-01T12:00:00.000Z","created_at":"2026-01-01T00:00:00.000Z","edited_at":%q,"_deleted":false}`, id, artist, title, edited))
}

func changed(t *testing.T, raw json.RawMessage, fields map[string]any) json.RawMessage {
	t.Helper()

	var doc map[string]any
	if e := json.Unmarshal(raw, &doc); e != nil {
		t.Fatal(e)
	}

	maps.Copy(doc, fields)

	out, e := json.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}

	return out
}

func push(t *testing.T, h *Handler, kind, token string, cp LibraryCheckpoint, rows ...LibraryChange) *httptest.ResponseRecorder {
	return syncCall(t, h, kind, "push", token, PushLibraryRequest{Epoch: cp.Epoch, Rows: rows})
}

const testArtistID = "10000000-0000-4000-8000-000000000001"
const testAlbumID = "20000000-0000-4000-8000-000000000001"
const stamp1 = "2026-01-01T00:00:01.000Z"
const stamp2 = "2026-01-01T00:00:02.000Z"
const stamp3 = "2026-01-01T00:00:03.000Z"

func TestReplicationLifecycleAndIsolation(t *testing.T) {
	h := testHandler(t)
	a := register(t, h, "a@sync.test")
	b := register(t, h, "b@sync.test")
	cp := initialPull(t, h, "artists", a.AccessToken)
	bcp := initialPull(t, h, "artists", b.AccessToken)
	status(t, syncCall(t, h, "artists", "pull", "", PullLibraryRequest{Limit: 10}), 401)
	artist := artistDocument(testArtistID, "Artist", stamp1)
	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: artist}), 200)
	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: artist}), 200)

	var seq int
	var count int
	if e := h.Database.QueryRow(`SELECT count(*),max(change_seq) FROM library_artists`).Scan(&count, &seq); e != nil || count != 1 || seq != 1 {
		t.Fatalf("duplicate write: %d %d %v", count, seq, e)
	}

	w := syncCall(t, h, "artists", "pull", b.AccessToken, PullLibraryRequest{Limit: 10})
	status(t, w, 200)

	var page struct {
		Documents  []json.RawMessage `json:"documents"`
		Checkpoint LibraryCheckpoint `json:"checkpoint"`
	}

	json.Unmarshal(w.Body.Bytes(), &page)
	if len(page.Documents) != 0 {
		t.Fatal("account data leaked")
	}

	w = push(t, h, "artists", b.AccessToken, bcp, LibraryChange{New: changed(t, artist, map[string]any{"name": "Intruder"})})
	if w.Code == 200 {
		t.Fatal("cross-account UUID acknowledged")
	}

	album := albumDocument(testAlbumID, testArtistID, "Album", stamp1)
	acp := initialPull(t, h, "albums", a.AccessToken)
	status(t, push(t, h, "albums", a.AccessToken, acp, LibraryChange{New: album}), 200)
	status(t, push(t, h, "albums", b.AccessToken, bcp, LibraryChange{New: albumDocument("20000000-0000-4000-8000-000000000002", testArtistID, "Other", stamp1)}), 409)
	updated := changed(t, artist, map[string]any{"name": "Newer", "edited_at": stamp2})
	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{Assumed: artist, New: updated}), 200)
	w = push(t, h, "artists", a.AccessToken, cp, LibraryChange{Assumed: artist, New: changed(t, artist, map[string]any{"name": "Competing", "edited_at": stamp3})})
	status(t, w, 200)
	conflicts := decode[[]json.RawMessage](t, w)
	if len(conflicts) != 1 || !equivalentJSON(conflicts[0], updated) {
		t.Fatalf("wrong conflict %s", w.Body)
	}

	w = push(t, h, "artists", a.AccessToken, cp, LibraryChange{Assumed: updated, New: changed(t, updated, map[string]any{"name": "Equal timestamp"})})
	status(t, w, 200)
	if len(decode[[]json.RawMessage](t, w)) != 1 {
		t.Fatal("equal timestamp replaced server")
	}

	deletedArtist := changed(t, updated, map[string]any{"_deleted": true, "edited_at": stamp3})
	w = push(t, h, "artists", a.AccessToken, cp, LibraryChange{Assumed: updated, New: deletedArtist})
	status(t, w, 409)
	dependencyFailure := decode[struct {
		Failures []LibraryChangeFailure `json:"failures"`
	}](t, w)
	if len(dependencyFailure.Failures) != 1 || !dependencyFailure.Failures[0].Retryable {
		t.Fatalf("dependency failure should be retryable: %s", w.Body)
	}
	deletedAlbum := changed(t, album, map[string]any{"_deleted": true, "edited_at": stamp2})
	status(t, push(t, h, "albums", a.AccessToken, acp, LibraryChange{Assumed: album, New: deletedAlbum}), 200)
	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{Assumed: updated, New: deletedArtist}), 200)
	w = syncCall(t, h, "albums", "pull", a.AccessToken, PullLibraryRequest{Limit: 10})
	status(t, w, 200)
	json.Unmarshal(w.Body.Bytes(), &page)

	var tombstone map[string]any
	json.Unmarshal(page.Documents[0], &tombstone)
	if tombstone["_deleted"] != true {
		t.Fatal("deletion missing from pull")
	}

	restoredAlbum := changed(t, deletedAlbum, map[string]any{"_deleted": false, "edited_at": stamp3})
	status(t, push(t, h, "albums", a.AccessToken, acp, LibraryChange{Assumed: deletedAlbum, New: restoredAlbum}), 409)
	restoredArtist := changed(t, deletedArtist, map[string]any{"_deleted": false, "edited_at": "2026-01-01T00:00:04.000Z"})
	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{Assumed: deletedArtist, New: restoredArtist}), 200)
	status(t, push(t, h, "albums", a.AccessToken, acp, LibraryChange{Assumed: deletedAlbum, New: restoredAlbum}), 200)
	status(t, request(h, h.DeleteAccount, "DELETE", "/api/account", `{"password":"password"}`, a.AccessToken, nil), 200)
	if e := h.Database.QueryRow(`SELECT (SELECT count(*) FROM library_artists WHERE user_id=$1) + (SELECT count(*) FROM library_albums WHERE user_id=$1)`, a.UserID).Scan(&count); e != nil || count != 0 {
		t.Fatalf("account records remain %v", e)
	}

	status(t, syncCall(t, h, "artists", "pull", a.AccessToken, PullLibraryRequest{Limit: 10}), 401)
}

func TestReplicationPaginationValidationAndPartialBatch(t *testing.T) {
	h := testHandler(t)
	a := register(t, h, "owner@sync.test")
	cp := initialPull(t, h, "artists", a.AccessToken)
	rows := []LibraryChange{}
	for i := 1; i <= 5; i++ {
		rows = append(rows, LibraryChange{New: artistDocument(fmt.Sprintf("30000000-0000-4000-8000-%012d", i), fmt.Sprintf("Artist %d", i), stamp1)})
	}

	status(t, push(t, h, "artists", a.AccessToken, cp, rows...), 200)
	seen := map[string]bool{}

	var checkpoint *LibraryCheckpoint
	for {
		w := syncCall(t, h, "artists", "pull", a.AccessToken, PullLibraryRequest{Limit: 2, Checkpoint: checkpoint})
		status(t, w, 200)

		var page struct {
			Documents []struct {
				ID string `json:"id"`
			} `json:"documents"`
			Checkpoint LibraryCheckpoint `json:"checkpoint"`
		}

		json.Unmarshal(w.Body.Bytes(), &page)
		for _, doc := range page.Documents {
			if seen[doc.ID] {
				t.Fatal("duplicate page")
			}

			seen[doc.ID] = true
		}

		if len(page.Documents) == 0 {
			if checkpoint == nil || *checkpoint != page.Checkpoint {
				t.Fatal("empty page advanced checkpoint")
			}

			break
		}

		checkpoint = &page.Checkpoint
	}

	if len(seen) != 5 {
		t.Fatalf("lost page: %d", len(seen))
	}

	future := cp
	future.Sequence = "999999999999"
	status(t, syncCall(t, h, "artists", "pull", a.AccessToken, PullLibraryRequest{Limit: 10, Checkpoint: &future}), 409)
	wrong := cp
	wrong.Epoch = "99999999-0000-4000-8000-000000000000"
	status(t, syncCall(t, h, "artists", "pull", a.AccessToken, PullLibraryRequest{Limit: 10, Checkpoint: &wrong}), 409)
	status(t, push(t, h, "artists", a.AccessToken, wrong, rows[0]), 409)
	status(t, syncCall(t, h, "artists", "pull", a.AccessToken, PullLibraryRequest{Limit: 0}), 400)
	for _, field := range []string{"id", "edited_at", "created_at", "_deleted", "name"} {
		bad := changed(t, rows[0].New, map[string]any{field: nil})
		status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: bad}), 422)
	}

	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: changed(t, rows[0].New, map[string]any{"owner_id": 99})}), 422)
	good := artistDocument("40000000-0000-4000-8000-000000000001", "Valid parent", stamp1)
	duplicate := artistDocument("40000000-0000-4000-8000-000000000002", "ARTIST 1", stamp1)
	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: duplicate}, LibraryChange{New: good}), 422)

	var count int
	if e := h.Database.QueryRow(`SELECT count(*) FROM library_artists WHERE id=$1`, "40000000-0000-4000-8000-000000000001").Scan(&count); e != nil || count != 1 {
		t.Fatalf("valid parent blocked by failed sibling: %v", e)
	}

	status(t, push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: good}), 200)

	var before int
	h.Database.QueryRow(`SELECT library_sync_sequence FROM users WHERE id=$1`, a.UserID).Scan(&before)
	bad := artistDocument("40000000-0000-4000-8000-000000000003", "Artist 1", stamp1)
	w := push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: bad})
	status(t, w, 422)
	invalidFailure := decode[struct {
		Failures []LibraryChangeFailure `json:"failures"`
	}](t, w)
	if len(invalidFailure.Failures) != 1 || invalidFailure.Failures[0].Retryable {
		t.Fatalf("invalid record should not be retryable: %s", w.Body)
	}

	var after int
	h.Database.QueryRow(`SELECT library_sync_sequence FROM users WHERE id=$1`, a.UserID).Scan(&after)
	if before != after {
		t.Fatal("rejected row published sequence")
	}
}

func TestReplicationCommitOrderAndConcurrentWrites(t *testing.T) {
	h := testHandler(t)
	a := register(t, h, "concurrent@sync.test")
	cp := initialPull(t, h, "artists", a.AccessToken)
	tx, e := h.Database.Begin()
	if e != nil {
		t.Fatal(e)
	}

	defer tx.Rollback()
	if _, e = models.LockSyncState(tx, a.UserID); e != nil {
		t.Fatal(e)
	}

	first := artistDocument(testArtistID, "Held transaction", stamp1)
	doc, e := validateDocument(first, models.ArtistKind)
	if e != nil {
		t.Fatal(e)
	}

	if _, _, e = applySyncRow(tx, a.UserID, models.ArtistKind, doc, nil); e != nil {
		t.Fatal(e)
	}

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- syncCall(t, h, "artists", "pull", a.AccessToken, PullLibraryRequest{Limit: 10}) }()
	select {
	case <-done:
		t.Fatal("pull passed an uncommitted writer")
	case <-time.After(100 * time.Millisecond):
	}

	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}

	select {
	case w := <-done:
		status(t, w, 200)

		var page struct {
			Documents []json.RawMessage `json:"documents"`
		}

		json.Unmarshal(w.Body.Bytes(), &page)
		if len(page.Documents) != 1 {
			t.Fatal("committed row skipped")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pull stuck")
	}

	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			responses <- push(t, h, "artists", a.AccessToken, cp, LibraryChange{New: first})
		}()
	}

	wg.Wait()
	close(responses)
	for w := range responses {
		status(t, w, 200)
	}

	var seq int
	if e = h.Database.QueryRow(`SELECT library_sync_sequence FROM users WHERE id=$1`, a.UserID).Scan(&seq); e != nil || seq != 1 {
		t.Fatalf("concurrent duplicate delivery advanced cursor %d %v", seq, e)
	}
}

func TestTypedDocumentValidation(t *testing.T) {
	for _, fixture := range []struct {
		kind models.LibraryKind
		raw  json.RawMessage
	}{
		{models.ArtistKind, artistDocument(testArtistID, "Artist", stamp1)},
		{models.AlbumKind, albumDocument(testAlbumID, testArtistID, "Album", stamp1)},
	} {
		t.Run(fmt.Sprint(fixture.kind), func(t *testing.T) {
			doc, err := validateDocument(fixture.raw, fixture.kind)
			if err != nil || !equivalentJSON(doc.Fields().Raw, fixture.raw) {
				t.Fatalf("valid document changed or rejected: %v", err)
			}

			encoded, err := json.Marshal(doc)
			if err != nil || !equivalentJSON(encoded, fixture.raw) {
				t.Fatalf("shared model changed the wire format: %s %v", encoded, err)
			}

			switch fixture.kind {
			case models.ArtistKind:
				if _, ok := doc.(*models.Artist); !ok {
					t.Fatal("artist validation did not return the storage model")
				}
			case models.AlbumKind:
				if _, ok := doc.(*models.Album); !ok {
					t.Fatal("album validation did not return the storage model")
				}
			}

			var fields map[string]json.RawMessage
			if err := json.Unmarshal(fixture.raw, &fields); err != nil {
				t.Fatal(err)
			}

			for name, value := range fields {
				delete(fields, name)
				missing, _ := json.Marshal(fields)
				if _, err := validateDocument(missing, fixture.kind); err == nil {
					t.Errorf("accepted missing %s", name)
				}
				fields[name] = value

				if _, err := validateDocument(changed(t, fixture.raw, map[string]any{name: []int{1}}), fixture.kind); err == nil {
					t.Errorf("accepted incorrect type for %s", name)
				}
			}
		})
	}

	album := albumDocument(testAlbumID, testArtistID, "Album", stamp1)
	for _, year := range []int64{-2147483649, 2147483648} {
		if _, err := validateAlbumDocument(changed(t, album, map[string]any{"year": year})); err == nil {
			t.Errorf("accepted year outside int32: %d", year)
		}
	}

	for _, rating := range []int{0, 11} {
		if _, err := validateAlbumDocument(changed(t, album, map[string]any{"rating": rating})); err == nil {
			t.Errorf("accepted invalid rating: %d", rating)
		}
	}

	empty := changed(t, album, map[string]any{
		"year": nil, "rating": nil, "comment": nil, "listened_at": nil, "listened": false,
	})
	if _, err := validateAlbumDocument(empty); err != nil {
		t.Fatalf("rejected nullable album fields: %v", err)
	}
}

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

	"github.com/Chanadu/better-music/models"
)

func syncCall(test *testing.T, handler *Handler, kind, op, identity string, body any) *httptest.ResponseRecorder {
	test.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		test.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sync/{kind}/pull", handler.PullLibrary)
	mux.HandleFunc("POST /api/sync/{kind}/push", handler.PushLibrary)
	httpRequest := httptest.NewRequest("POST", "/api/sync/"+kind+"/"+op, bytes.NewReader(raw))
	recorder := httptest.NewRecorder()
	serveIdentity(handler, mux, recorder, httpRequest, identity)
	return recorder
}

func initialPull(test *testing.T, handler *Handler, kind, identity string) LibraryCheckpoint {
	recorder := syncCall(test, handler, kind, "pull", identity, PullLibraryRequest{Limit: 100})
	status(test, recorder, 200)

	var page struct {
		Checkpoint LibraryCheckpoint `json:"checkpoint"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		test.Fatal(err)
	}

	return page.Checkpoint
}

func artistDocument(id, name, edited string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"id":%q,"name":%q,"cover_url":null,"spotify_id":null,"created_at":"2026-01-01T00:00:00.000Z","edited_at":%q,"_deleted":false}`, id, name, edited))
}

func albumDocument(id, artist, title, edited string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"id":%q,"artist_id":%q,"title":%q,"cover_url":null,"spotify_id":null,"year":2024,"listened":true,"rating":8,"comment":"Keep these notes","listened_at":"2026-01-01T12:00:00.000Z","created_at":"2026-01-01T00:00:00.000Z","edited_at":%q,"_deleted":false}`, id, artist, title, edited))
}

func changed(test *testing.T, raw json.RawMessage, fields map[string]any) json.RawMessage {
	test.Helper()

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		test.Fatal(err)
	}

	maps.Copy(doc, fields)

	out, err := json.Marshal(doc)
	if err != nil {
		test.Fatal(err)
	}

	return out
}

func push(test *testing.T, handler *Handler, kind, identity string, cp LibraryCheckpoint, rows ...LibraryChange) *httptest.ResponseRecorder {
	return syncCall(test, handler, kind, "push", identity, PushLibraryRequest{Epoch: cp.Epoch, Rows: rows})
}

const testArtistID = "10000000-0000-4000-8000-000000000001"
const testAlbumID = "20000000-0000-4000-8000-000000000001"
const stamp1 = "2026-01-01T00:00:01.000Z"
const stamp2 = "2026-01-01T00:00:02.000Z"
const stamp3 = "2026-01-01T00:00:03.000Z"

func TestReplicationLifecycleAndIsolation(test *testing.T) {
	handler := testHandler(test)
	firstAccount := createAccount(test, handler, "a@sync.test")
	secondAccount := createAccount(test, handler, "b@sync.test")
	cp := initialPull(test, handler, "artists", firstAccount.Identity)
	bcp := initialPull(test, handler, "artists", secondAccount.Identity)
	status(test, syncCall(test, handler, "artists", "pull", "", PullLibraryRequest{Limit: 10}), 401)
	artist := artistDocument(testArtistID, "Artist", stamp1)
	status(test, push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{New: artist}), 200)
	status(test, push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{New: artist}), 200)

	var seq int
	var count int
	if err := handler.Database.QueryRow(`SELECT count(*),max(change_seq) FROM library_artists`).Scan(&count, &seq); err != nil || count != 1 || seq != 1 {
		test.Fatalf("duplicate write: %d %d %v", count, seq, err)
	}

	recorder := syncCall(test, handler, "artists", "pull", secondAccount.Identity, PullLibraryRequest{Limit: 10})
	status(test, recorder, 200)

	var page struct {
		Documents  []json.RawMessage `json:"documents"`
		Checkpoint LibraryCheckpoint `json:"checkpoint"`
	}

	json.Unmarshal(recorder.Body.Bytes(), &page)
	if len(page.Documents) != 0 {
		test.Fatal("account data leaked")
	}

	recorder = push(test, handler, "artists", secondAccount.Identity, bcp, LibraryChange{New: changed(test, artist, map[string]any{"name": "Intruder"})})
	if recorder.Code == 200 {
		test.Fatal("cross-account UUID acknowledged")
	}

	album := albumDocument(testAlbumID, testArtistID, "Album", stamp1)
	acp := initialPull(test, handler, "albums", firstAccount.Identity)
	status(test, push(test, handler, "albums", firstAccount.Identity, acp, LibraryChange{New: album}), 200)
	status(test, push(test, handler, "albums", secondAccount.Identity, bcp, LibraryChange{New: albumDocument("20000000-0000-4000-8000-000000000002", testArtistID, "Other", stamp1)}), 409)
	updated := changed(test, artist, map[string]any{"name": "Newer", "edited_at": stamp2})
	status(test, push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{Assumed: artist, New: updated}), 200)
	recorder = push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{Assumed: artist, New: changed(test, artist, map[string]any{"name": "Competing", "edited_at": stamp3})})
	status(test, recorder, 200)
	conflicts := decode[[]json.RawMessage](test, recorder)
	if len(conflicts) != 1 || !equivalentJSON(conflicts[0], updated) {
		test.Fatalf("wrong conflict %s", recorder.Body)
	}

	recorder = push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{Assumed: updated, New: changed(test, updated, map[string]any{"name": "Equal timestamp"})})
	status(test, recorder, 200)
	if len(decode[[]json.RawMessage](test, recorder)) != 1 {
		test.Fatal("equal timestamp replaced server")
	}

	deletedArtist := changed(test, updated, map[string]any{"_deleted": true, "edited_at": stamp3})
	recorder = push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{Assumed: updated, New: deletedArtist})
	status(test, recorder, 409)
	dependencyFailure := decode[struct {
		Failures []LibraryChangeFailure `json:"failures"`
	}](test, recorder)
	if len(dependencyFailure.Failures) != 1 || !dependencyFailure.Failures[0].Retryable {
		test.Fatalf("dependency failure should be retryable: %s", recorder.Body)
	}
	deletedAlbum := changed(test, album, map[string]any{"_deleted": true, "edited_at": stamp2})
	status(test, push(test, handler, "albums", firstAccount.Identity, acp, LibraryChange{Assumed: album, New: deletedAlbum}), 200)
	status(test, push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{Assumed: updated, New: deletedArtist}), 200)
	recorder = syncCall(test, handler, "albums", "pull", firstAccount.Identity, PullLibraryRequest{Limit: 10})
	status(test, recorder, 200)
	json.Unmarshal(recorder.Body.Bytes(), &page)

	var tombstone map[string]any
	json.Unmarshal(page.Documents[0], &tombstone)
	if tombstone["_deleted"] != true {
		test.Fatal("deletion missing from pull")
	}

	restoredAlbum := changed(test, deletedAlbum, map[string]any{"_deleted": false, "edited_at": stamp3})
	status(test, push(test, handler, "albums", firstAccount.Identity, acp, LibraryChange{Assumed: deletedAlbum, New: restoredAlbum}), 409)
	restoredArtist := changed(test, deletedArtist, map[string]any{"_deleted": false, "edited_at": "2026-01-01T00:00:04.000Z"})
	status(test, push(test, handler, "artists", firstAccount.Identity, cp, LibraryChange{Assumed: deletedArtist, New: restoredArtist}), 200)
	status(test, push(test, handler, "albums", firstAccount.Identity, acp, LibraryChange{Assumed: deletedAlbum, New: restoredAlbum}), 200)
	// Account lifecycle HTTP behavior is covered by the real-Core suite;
	// this assertion keeps replication's ownership/cascade coverage independent.
	if _, err := handler.Database.Exec(`DELETE FROM users WHERE id=$1`, firstAccount.UserID); err != nil {
		test.Fatal(err)
	}
	if err := handler.Database.QueryRow(`SELECT (SELECT count(*) FROM library_artists WHERE user_id=$1) + (SELECT count(*) FROM library_albums WHERE user_id=$1)`, firstAccount.UserID).Scan(&count); err != nil || count != 0 {
		test.Fatalf("account records remain %v", err)
	}

	status(test, syncCall(test, handler, "artists", "pull", firstAccount.Identity, PullLibraryRequest{Limit: 10}), 403)
}

func TestReplicationPaginationValidationAndPartialBatch(test *testing.T) {
	handler := testHandler(test)
	account := createAccount(test, handler, "owner@sync.test")
	cp := initialPull(test, handler, "artists", account.Identity)
	rows := []LibraryChange{}
	for index := 1; index <= 5; index++ {
		rows = append(rows, LibraryChange{New: artistDocument(fmt.Sprintf("30000000-0000-4000-8000-%012d", index), fmt.Sprintf("Artist %d", index), stamp1)})
	}

	status(test, push(test, handler, "artists", account.Identity, cp, rows...), 200)
	seen := map[string]bool{}

	var checkpoint *LibraryCheckpoint
	for {
		recorder := syncCall(test, handler, "artists", "pull", account.Identity, PullLibraryRequest{Limit: 2, Checkpoint: checkpoint})
		status(test, recorder, 200)

		var page struct {
			Documents []struct {
				ID string `json:"id"`
			} `json:"documents"`
			Checkpoint LibraryCheckpoint `json:"checkpoint"`
		}

		json.Unmarshal(recorder.Body.Bytes(), &page)
		for _, doc := range page.Documents {
			if seen[doc.ID] {
				test.Fatal("duplicate page")
			}

			seen[doc.ID] = true
		}

		if len(page.Documents) == 0 {
			if checkpoint == nil || *checkpoint != page.Checkpoint {
				test.Fatal("empty page advanced checkpoint")
			}

			break
		}

		checkpoint = &page.Checkpoint
	}

	if len(seen) != 5 {
		test.Fatalf("lost page: %d", len(seen))
	}

	future := cp
	future.Sequence = "999999999999"
	status(test, syncCall(test, handler, "artists", "pull", account.Identity, PullLibraryRequest{Limit: 10, Checkpoint: &future}), 409)
	wrong := cp
	wrong.Epoch = "99999999-0000-4000-8000-000000000000"
	status(test, syncCall(test, handler, "artists", "pull", account.Identity, PullLibraryRequest{Limit: 10, Checkpoint: &wrong}), 409)
	status(test, push(test, handler, "artists", account.Identity, wrong, rows[0]), 409)
	status(test, syncCall(test, handler, "artists", "pull", account.Identity, PullLibraryRequest{Limit: 0}), 400)
	for _, field := range []string{"id", "edited_at", "created_at", "_deleted", "name"} {
		bad := changed(test, rows[0].New, map[string]any{field: nil})
		status(test, push(test, handler, "artists", account.Identity, cp, LibraryChange{New: bad}), 422)
	}

	status(test, push(test, handler, "artists", account.Identity, cp, LibraryChange{New: changed(test, rows[0].New, map[string]any{"owner_id": 99})}), 422)
	good := artistDocument("40000000-0000-4000-8000-000000000001", "Valid parent", stamp1)
	duplicate := artistDocument("40000000-0000-4000-8000-000000000002", "ARTIST 1", stamp1)
	status(test, push(test, handler, "artists", account.Identity, cp, LibraryChange{New: duplicate}, LibraryChange{New: good}), 422)

	var count int
	if err := handler.Database.QueryRow(`SELECT count(*) FROM library_artists WHERE id=$1`, "40000000-0000-4000-8000-000000000001").Scan(&count); err != nil || count != 1 {
		test.Fatalf("valid parent blocked by failed sibling: %v", err)
	}

	status(test, push(test, handler, "artists", account.Identity, cp, LibraryChange{New: good}), 200)

	var before int
	handler.Database.QueryRow(`SELECT library_sync_sequence FROM users WHERE id=$1`, account.UserID).Scan(&before)
	bad := artistDocument("40000000-0000-4000-8000-000000000003", "Artist 1", stamp1)
	recorder := push(test, handler, "artists", account.Identity, cp, LibraryChange{New: bad})
	status(test, recorder, 422)
	invalidFailure := decode[struct {
		Failures []LibraryChangeFailure `json:"failures"`
	}](test, recorder)
	if len(invalidFailure.Failures) != 1 || invalidFailure.Failures[0].Retryable {
		test.Fatalf("invalid record should not be retryable: %s", recorder.Body)
	}

	var after int
	handler.Database.QueryRow(`SELECT library_sync_sequence FROM users WHERE id=$1`, account.UserID).Scan(&after)
	if before != after {
		test.Fatal("rejected row published sequence")
	}
}

func TestReplicationCommitOrderAndConcurrentWrites(test *testing.T) {
	handler := testHandler(test)
	account := createAccount(test, handler, "concurrent@sync.test")
	cp := initialPull(test, handler, "artists", account.Identity)
	tx, err := handler.Database.Begin()
	if err != nil {
		test.Fatal(err)
	}

	defer tx.Rollback()
	if _, err = models.LockSyncState(tx, account.UserID); err != nil {
		test.Fatal(err)
	}

	first := artistDocument(testArtistID, "Held transaction", stamp1)
	doc, err := validateDocument(first, models.ArtistKind)
	if err != nil {
		test.Fatal(err)
	}

	if _, _, err = applySyncRow(tx, account.UserID, models.ArtistKind, doc, nil); err != nil {
		test.Fatal(err)
	}

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- syncCall(test, handler, "artists", "pull", account.Identity, PullLibraryRequest{Limit: 10})
	}()
	select {
	case <-done:
		test.Fatal("pull passed an uncommitted writer")
	case <-time.After(100 * time.Millisecond):
	}

	if err = tx.Commit(); err != nil {
		test.Fatal(err)
	}

	select {
	case recorder := <-done:
		status(test, recorder, 200)

		var page struct {
			Documents []json.RawMessage `json:"documents"`
		}

		json.Unmarshal(recorder.Body.Bytes(), &page)
		if len(page.Documents) != 1 {
			test.Fatal("committed row skipped")
		}
	case <-time.After(5 * time.Second):
		test.Fatal("pull stuck")
	}

	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 8)
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			responses <- push(test, handler, "artists", account.Identity, cp, LibraryChange{New: first})
		}()
	}

	wg.Wait()
	close(responses)
	for recorder := range responses {
		status(test, recorder, 200)
	}

	var seq int
	if err = handler.Database.QueryRow(`SELECT library_sync_sequence FROM users WHERE id=$1`, account.UserID).Scan(&seq); err != nil || seq != 1 {
		test.Fatalf("concurrent duplicate delivery advanced cursor %d %v", seq, err)
	}
}

func TestTypedDocumentValidation(test *testing.T) {
	for _, fixture := range []struct {
		kind models.LibraryKind
		raw  json.RawMessage
	}{
		{models.ArtistKind, artistDocument(testArtistID, "Artist", stamp1)},
		{models.AlbumKind, albumDocument(testAlbumID, testArtistID, "Album", stamp1)},
	} {
		test.Run(fmt.Sprint(fixture.kind), func(test *testing.T) {
			doc, err := validateDocument(fixture.raw, fixture.kind)
			if err != nil || !equivalentJSON(doc.Fields().Raw, fixture.raw) {
				test.Fatalf("valid document changed or rejected: %v", err)
			}

			encoded, err := json.Marshal(doc)
			if err != nil || !equivalentJSON(encoded, fixture.raw) {
				test.Fatalf("shared model changed the wire format: %s %v", encoded, err)
			}

			switch fixture.kind {
			case models.ArtistKind:
				if _, ok := doc.(*models.Artist); !ok {
					test.Fatal("artist validation did not return the storage model")
				}
			case models.AlbumKind:
				if _, ok := doc.(*models.Album); !ok {
					test.Fatal("album validation did not return the storage model")
				}
			}

			var fields map[string]json.RawMessage
			if err := json.Unmarshal(fixture.raw, &fields); err != nil {
				test.Fatal(err)
			}

			for name, value := range fields {
				delete(fields, name)
				missing, _ := json.Marshal(fields)
				if _, err := validateDocument(missing, fixture.kind); err == nil {
					test.Errorf("accepted missing %s", name)
				}
				fields[name] = value

				if _, err := validateDocument(changed(test, fixture.raw, map[string]any{name: []int{1}}), fixture.kind); err == nil {
					test.Errorf("accepted incorrect type for %s", name)
				}
			}
		})
	}

	album := albumDocument(testAlbumID, testArtistID, "Album", stamp1)
	for _, year := range []int64{-2147483649, 2147483648} {
		if _, err := validateAlbumDocument(changed(test, album, map[string]any{"year": year})); err == nil {
			test.Errorf("accepted year outside int32: %d", year)
		}
	}

	for _, rating := range []int{0, 11} {
		if _, err := validateAlbumDocument(changed(test, album, map[string]any{"rating": rating})); err == nil {
			test.Errorf("accepted invalid rating: %d", rating)
		}
	}

	empty := changed(test, album, map[string]any{
		"year": nil, "rating": nil, "comment": nil, "listened_at": nil, "listened": false,
	})
	if _, err := validateAlbumDocument(empty); err != nil {
		test.Fatalf("rejected nullable album fields: %v", err)
	}
}

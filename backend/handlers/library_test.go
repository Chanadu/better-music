package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Chanadu/better-music/models"
)

func TestLibraryLifecycle(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "owner@example.com")
	otherAccountTokens := register(test, apiHandler, "other@example.com")
	token := accountTokens.AccessToken

	for _, endpointHandler := range []http.HandlerFunc{apiHandler.GetArtists, apiHandler.GetAlbums} {
		responseRecorder := request(apiHandler, endpointHandler, "GET", "/", "", token, nil)

		status(test, responseRecorder, 200)
		if responseRecorder.Body.String() != "[]\n" {
			test.Fatal(responseRecorder.Body)
		}
	}

	responseRecorder := request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", `{"name":" Artist ","cover_url":"image","spotify_id":"spotify"}`, token, nil)
	status(test, responseRecorder, 201)

	artist := decode[models.Artist](test, responseRecorder)
	path := fmt.Sprintf("/api/artists/%d", artist.ID)
	if artist.Name != "Artist" || artist.Version != 1 || responseRecorder.Header().Get("ETag") != `"1"` {
		test.Fatal(responseRecorder.Body)
	}

	status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", `{"name":"artist"}`, token, nil), 409)

	status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", `{"name":"artist"}`, otherAccountTokens.AccessToken, nil), 201)

	status(test, request(apiHandler, apiHandler.GetArtist, "GET", path, "", otherAccountTokens.AccessToken, nil), 404)

	status(test, request(apiHandler, apiHandler.UpdateArtist, "PUT", path, `{"name":"stolen"}`, otherAccountTokens.AccessToken, nil), 404)

	status(test, request(apiHandler, apiHandler.DeleteArtist, "DELETE", path, "", otherAccountTokens.AccessToken, nil), 200)

	responseRecorder = request(apiHandler, apiHandler.GetArtist, "GET", path, "", token, nil)
	status(test, responseRecorder, 200)
	if decode[models.Artist](test, responseRecorder).Name != "Artist" {
		test.Fatal("other user changed artist")
	}

	status(test, request(apiHandler, apiHandler.GetArtistAlbums, "GET", path+"/albums", "", token, nil), 200)

	responseRecorder = request(
		apiHandler,
		apiHandler.UpdateArtist,
		"PUT",
		path,
		`{"name":"Renamed","cover_url":null,"spotify_id":null}`,
		token,
		map[string]string{"If-Match": `"1"`},
	)
	status(test, responseRecorder, 200)

	artist = decode[models.Artist](test, responseRecorder)
	if artist.Version != 2 || artist.CoverURL != nil || artist.SpotifyID != nil || artist.Name != "Renamed" {
		test.Fatal(responseRecorder.Body)
	}

	responseRecorder = request(apiHandler, apiHandler.UpdateArtist, "PUT", path, `{"name":"stale"}`, token, map[string]string{"If-Match": "1"})
	status(test, responseRecorder, 412)
	if decode[ArtistConflictResponse](test, responseRecorder).Current.Version != 2 || responseRecorder.Header().Get("ETag") != `"2"` {
		test.Fatal(responseRecorder.Body)
	}

	status(test, request(apiHandler, apiHandler.DeleteArtist, "DELETE", path, "", token, map[string]string{"If-Match": "1"}), 412)

	status(test, request(apiHandler, apiHandler.UpdateArtist, "PUT", path, `{}`, token, nil), 400)

	body := fmt.Sprintf(`{"artist_id":%d,"title":" Album ","year":2001,"cover_url":"cover","spotify_id":"id","listened":true,"rating":9,"comment":"great","listened_at":"2025-01-01"}`, artist.ID)
	responseRecorder = request(apiHandler, apiHandler.CreateAlbum, "POST", "/api/albums", body, token, nil)
	status(test, responseRecorder, 201)

	album := decode[models.Album](test, responseRecorder)
	albumPath := fmt.Sprintf("/api/albums/%d", album.ID)
	if album.Title != "Album" || album.Rating == nil || *album.Rating != 9 || album.Version != 1 {
		test.Fatal(responseRecorder.Body)
	}

	status(test, request(apiHandler, apiHandler.CreateAlbum, "POST", "/api/albums", strings.Replace(body, " Album ", "album", 1), token, nil), 409)

	status(test, request(apiHandler, apiHandler.CreateAlbum, "POST", "/api/albums", body, otherAccountTokens.AccessToken, nil), 400)

	responseRecorder = request(apiHandler, apiHandler.GetAlbum, "GET", albumPath+fmt.Sprintf("?artist_id=%d", artist.ID), "", token, nil)
	status(test, responseRecorder, 200)
	if responseRecorder.Header().Get("ETag") != `"1"` {
		test.Fatal(responseRecorder.Header())
	}

	status(test, request(apiHandler, apiHandler.GetAlbum, "GET", albumPath, fmt.Sprintf(`{"artist_id":%d}`, artist.ID), token, nil), 200)

	status(test, request(apiHandler, apiHandler.GetAlbums, "GET", "/api/albums", "", token, nil), 200)

	responseRecorder = request(apiHandler, apiHandler.GetArtistAlbums, "GET", path+"/albums", "", token, nil)
	status(test, responseRecorder, 200)
	if len(decode[[]models.Album](test, responseRecorder)) != 1 {
		test.Fatal(responseRecorder.Body)
	}

	status(
		test,
		request(apiHandler, apiHandler.GetAlbum, "GET", albumPath+fmt.Sprintf("?artist_id=%d", artist.ID), "", otherAccountTokens.AccessToken, nil),
		404,
	)

	status(test, request(apiHandler, apiHandler.UpdateAlbum, "PUT", albumPath, `{"title":"stolen"}`, otherAccountTokens.AccessToken, nil), 404)

	status(test, request(apiHandler, apiHandler.DeleteArtist, "DELETE", path, "", token, nil), 400)

	status(test, request(apiHandler, apiHandler.UpdateAlbum, "PUT", albumPath, `{"artist_id":999,"title":"wrong"}`, token, nil), 404)

	responseRecorder = request(
		apiHandler,
		apiHandler.UpdateAlbum,
		"PUT",
		albumPath,
		`{"listened":false,"rating":8,"listened_at":"2025-01-01","year":null,"comment":null}`,
		token,
		map[string]string{"If-Match": "1"},
	)
	status(test, responseRecorder, 200)

	album = decode[models.Album](test, responseRecorder)
	if album.Listened || album.Rating != nil || album.ListenedAt != nil || album.Year != nil || album.Comment != nil ||
		album.Version != 2 {
		test.Fatal(responseRecorder.Body)
	}

	responseRecorder = request(apiHandler, apiHandler.UpdateAlbum, "PUT", albumPath, `{"title":"stale"}`, token, map[string]string{"If-Match": "1"})
	status(test, responseRecorder, 412)
	if decode[AlbumConflictResponse](test, responseRecorder).Current.Version != 2 {
		test.Fatal(responseRecorder.Body)
	}

	status(test, request(apiHandler, apiHandler.DeleteAlbum, "DELETE", albumPath, `{"artist_id":999}`, token, nil), 404)

	status(test, request(apiHandler, apiHandler.DeleteAlbum, "DELETE", albumPath, "", token, map[string]string{"If-Match": "1"}), 412)

	status(test, request(apiHandler, apiHandler.DeleteAlbum, "DELETE", albumPath, "", token, map[string]string{"If-Match": "2"}), 200)

	status(test, request(apiHandler, apiHandler.DeleteAlbum, "DELETE", albumPath, "", token, nil), 200)

	status(test, request(apiHandler, apiHandler.GetAlbum, "GET", albumPath+fmt.Sprintf("?artist_id=%d", artist.ID), "", token, nil), 404)

	status(test, request(apiHandler, apiHandler.DeleteArtist, "DELETE", path, "", token, nil), 200)

	status(test, request(apiHandler, apiHandler.DeleteArtist, "DELETE", path, "", token, nil), 200)
}

func TestMutationIdempotency(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "a@example.com")
	otherAccountTokens := register(test, apiHandler, "b@example.com")
	headers := map[string]string{"Idempotency-Key": "create"}
	body := `{"name":"Artist"}`
	first := request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", body, accountTokens.AccessToken, headers)
	status(test, first, 201)

	replay := request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", body, accountTokens.AccessToken, headers)
	status(test, replay, 201)
	if first.Body.String() != replay.Body.String() || first.Header().Get("ETag") != replay.Header().Get("ETag") {
		test.Fatal("replay changed response")
	}

	status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", `{"name":"Different"}`, accountTokens.AccessToken, headers), 409)

	status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", body, otherAccountTokens.AccessToken, headers), 201)

	headers = map[string]string{"Idempotency-Key": "error"}
	first = request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", body, accountTokens.AccessToken, headers)
	status(test, first, 409)

	recordID := decode[models.Artist](test, replay).ID
	status(test, request(apiHandler, apiHandler.DeleteArtist, "DELETE", fmt.Sprintf("/api/artists/%d", recordID), "", accountTokens.AccessToken, nil), 200)

	replay = request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", body, accountTokens.AccessToken, headers)
	status(test, replay, 409)
	if first.Body.String() != replay.Body.String() {
		test.Fatal("error replay changed")
	}

	var count int
	if err := apiHandler.Database.QueryRow("SELECT count(*) FROM artists WHERE user_id=$1", accountTokens.UserID).Scan(&count); err != nil || count != 0 {
		test.Fatalf("count=%d %v", count, err)
	}

	for _, key := range []string{" ", strings.Repeat("x", 256)} {
		status(
			test,
			request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", body, accountTokens.AccessToken, map[string]string{"Idempotency-Key": key}),
			400,
		)
	}

	status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", strings.Repeat("x", (1<<20)+1), accountTokens.AccessToken, nil), 400)
}

func TestLibraryValidation(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "a@example.com")
	token := accountTokens.AccessToken

	for _, body := range []string{
		`{}`,
		`null`,
		`{"name":" "}`,
		`{"unexpected":1}`,
		`{"name":null}`,
	} {
		status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", body, token, nil), 400)
	}

	for _, body := range []string{
		`{}`,
		`{"title":"Album"}`,
		`{"artist_id":999,"title":"Album"}`,
		`{"artist_id":1,"rating":11}`,
	} {
		status(test, request(apiHandler, apiHandler.CreateAlbum, "POST", "/api/albums", body, token, nil), 400)
	}

	for _, recordID := range []string{"bad", "0", "-1", "2147483648"} {
		status(test, request(apiHandler, apiHandler.UpdateArtist, "PUT", "/api/artists/"+recordID, `{"name":"x"}`, token, nil), 400)
	}

	for _, endpointHandler := range []http.HandlerFunc{apiHandler.GetArtist, apiHandler.GetArtistAlbums} {
		status(test, request(apiHandler, endpointHandler, "GET", "/api/artists/bad", "", token, nil), 400)

		status(test, request(apiHandler, endpointHandler, "GET", "/api/artists/999", "", token, nil), 404)
	}

	for _, path := range []string{
		"/api/albums/1",
		"/api/albums/1?artist_id=0",
		"/api/albums/1?artist_id=-1",
	} {
		status(test, request(apiHandler, apiHandler.GetAlbum, "GET", path, "", token, nil), 400)
	}

	status(test, request(apiHandler, apiHandler.GetAlbum, "GET", "/api/albums/1?artist_id=999", "", token, nil), 404)

	status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", `{"name":"x"}`, token, map[string]string{"If-Match": "*"}), 400)
}

func TestConcurrentMutations(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "a@example.com")
	var waitGroup sync.WaitGroup
	results := make(chan string, 8)

	for attemptIndex := 0; attemptIndex < 8; attemptIndex++ {
		waitGroup.Go(func() {
			responseRecorder := request(
				apiHandler,
				apiHandler.CreateArtist,
				"POST",
				"/api/artists",
				`{"name":"Once"}`,
				accountTokens.AccessToken,
				map[string]string{"Idempotency-Key": "once"},
			)
			results <- fmt.Sprintf("%d:%s", responseRecorder.Code, responseRecorder.Body.String())
		})
	}

	waitGroup.Wait()
	close(results)
	first := ""

	for result := range results {
		if !strings.HasPrefix(result, "201:") {
			test.Fatal(result)
		}

		if first == "" {
			first = result
		} else if first != result {
			test.Fatal("concurrent replay differs")
		}
	}

	responseRecorder := request(apiHandler, apiHandler.GetArtists, "GET", "/api/artists", "", accountTokens.AccessToken, nil)
	artists := decode[[]models.Artist](test, responseRecorder)
	if len(artists) != 1 {
		test.Fatal(responseRecorder.Body)
	}

	codes := make(chan int, 2)

	for _, name := range []string{"First", "Second"} {
		waitGroup.Go(func() {
			codes <- request(
				apiHandler,
				apiHandler.UpdateArtist,
				"PUT",
				fmt.Sprintf("/api/artists/%d", artists[0].ID),
				fmt.Sprintf(`{"name":%q}`, name),
				accountTokens.AccessToken,
				map[string]string{"If-Match": "1"},
			).Code
		})
	}

	waitGroup.Wait()
	close(codes)
	counts := map[int]int{}

	for code := range codes {
		counts[code]++
	}

	if counts[200] != 1 || counts[412] != 1 {
		test.Fatalf("version race: %v", counts)
	}
}

func TestDatabaseUnavailable(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "a@example.com")
	if err := apiHandler.Database.Close(); err != nil {
		test.Fatal(err)
	}

	for _, endpointHandler := range []http.HandlerFunc{
		apiHandler.GetAccount,
		apiHandler.GetArtists,
		apiHandler.GetArtistAlbums,
		apiHandler.GetArtist,
		apiHandler.GetAlbums,
		apiHandler.CreateArtist,
		apiHandler.CreateAlbum,
	} {
		status(test, request(apiHandler, endpointHandler, "POST", "/api/artists/1", `{"name":"x"}`, accountTokens.AccessToken, nil), 500)
	}

	status(test, request(apiHandler, apiHandler.GetAlbum, "GET", "/api/albums/1?artist_id=1", "", accountTokens.AccessToken, nil), 500)

	status(test, request(apiHandler, apiHandler.AuthRegister, "POST", "/", `{"email":"x@example.com","password":"password"}`, "", nil), 500)

	status(test, request(apiHandler, apiHandler.AuthRefresh, "POST", "/", `{"refresh_token":"x"}`, "", nil), 500)

	status(test, request(apiHandler, apiHandler.AuthLogout, "POST", "/", `{"refresh_token":"x"}`, "", nil), 500)
}

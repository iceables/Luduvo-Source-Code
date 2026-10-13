package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
)

func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		Views: html.New("../../views", ".html"),
	})
	app.Use(session.New())
	Setup(app)
	return app
}

func TestStoreServesHTMLToBrowsers(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/store", nil)
	req.Header.Set("Accept", "text/html")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /store status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Lune Packs") {
		t.Fatalf("GET /store did not render the store page")
	}
}

func TestStorePreservesJSONCatalogRequests(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/store", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /store JSON status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("GET /store JSON content type = %q", resp.Header.Get("Content-Type"))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"items"`) {
		t.Fatalf("GET /store JSON response is missing catalog items")
	}
}

func TestCatalogAPIIsAvailable(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/store", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/store status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("GET /api/v1/store content type = %q", resp.Header.Get("Content-Type"))
	}
}

func TestCDNResolveAcceptsNumericAssetIDs(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodPost, "/cdn/resolve", strings.NewReader(`[544,72057594037930593]`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /cdn/resolve status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["544"] != "/assets/544" || result["72057594037930593"] != "/assets/72057594037930593" {
		t.Fatalf("POST /cdn/resolve result = %#v", result)
	}
}

func TestCatalogModelAssetRejectsUntrustedPaths(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/model-asset?path=https%3A%2F%2F127.0.0.1%2Fetc%2Fpasswd", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /model-asset with an untrusted path status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestInventoryAPIRequiresAuthentication(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/inventory status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestShopItemsFailsExplicitlyWhenServiceUnavailable(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shop/items", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/v1/shop/items status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestAdminAssetAPIPathsAreRegistered(t *testing.T) {
	app := newTestApp()
	for _, path := range []string{"/api/v1/assets/1/file", "/api/v1/assets/1/render"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusInternalServerError {
				t.Fatalf("GET %s status = %d, want %d (registered handler with no asset service)", path, resp.StatusCode, http.StatusInternalServerError)
			}
		})
	}
}

func TestCatalogThumbnailUsesImageMIMEType(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source file")
	}

	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/static/catalog/17.img", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET thumbnail status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("GET thumbnail content type = %q, want image/png", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("GET thumbnail did not return a PNG image")
	}
}

func TestGamesPageRendersPlayPageWithoutRedirectLoop(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/games", nil)
	req.Header.Set("Accept", "text/html")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /games status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Play") {
		t.Fatal("GET /games did not render the play page")
	}
}

func TestGameThumbnailUsesFallbackImageWhenNoUploadExists(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source file")
	}

	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	app := newTestApp()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/thumbnails/1.png", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /thumbnails/1.png status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("GET /thumbnails/1.png content type = %q, want image/png", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("GET /thumbnails/1.png did not return a PNG image")
	}
}

func TestCreatePlacesRedirectsToGameCreator(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/create/places", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET /create/places status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if got := resp.Header.Get("Location"); got != "/create?type=game" {
		t.Fatalf("GET /create/places redirect = %q, want /create?type=game", got)
	}
}

func TestPlaceDetailReportsUnavailableGameService(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/places/1", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /places/1 status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestPlaceServersReturnsFrontendCompatibleEmptyList(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/places/1/servers", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /places/1/servers status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var servers []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		t.Fatal(err)
	}
	if servers == nil || len(servers) != 0 {
		t.Fatalf("GET /places/1/servers returned %#v, want an empty server list", servers)
	}
}

func TestSafetyPageAPIsReturnFrontendCompatibleData(t *testing.T) {
	app := newTestApp()

	t.Run("legal documents", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/legal/current", nil))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /legal/current status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		var payload struct {
			Documents []struct {
				ID          string `json:"id"`
				Kind        string `json:"kind"`
				URL         string `json:"url"`
				PublishedAt int64  `json:"published_at"`
				EffectiveAt int64  `json:"effective_at"`
				Accepted    bool   `json:"accepted"`
			} `json:"documents"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Documents) == 0 {
			t.Fatal("GET /legal/current returned no documents")
		}
		for _, document := range payload.Documents {
			if document.ID == "" || document.Kind == "" || document.URL == "" || document.PublishedAt == 0 || document.EffectiveAt == 0 {
				t.Fatalf("GET /legal/current returned incomplete document: %+v", document)
			}
			if document.Accepted {
				t.Fatalf("anonymous legal document %q is marked accepted", document.ID)
			}
		}
	})

	t.Run("ticket categories", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ticket-categories", nil))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /ticket-categories status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		var payload struct {
			Categories []struct {
				Code  string `json:"code"`
				Label string `json:"label"`
			} `json:"categories"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Categories) == 0 {
			t.Fatal("GET /ticket-categories returned no categories")
		}
		for _, category := range payload.Categories {
			if category.Code == "" || category.Label == "" {
				t.Fatalf("GET /ticket-categories returned incomplete category: %+v", category)
			}
		}
	})

	t.Run("legal acceptance requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/me/legal/accept", nil)
		req.Header.Set("Accept", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("POST /me/legal/accept status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})
}

func TestGuidelineSectionsReturnsSafetyReportCategories(t *testing.T) {
	app := newTestApp()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/guideline-sections", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /guideline-sections status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var payload struct {
		Sections []struct {
			Code   string `json:"code"`
			Label  string `json:"label"`
			Corpus string `json:"corpus"`
			Sort   int    `json:"sort"`
		} `json:"sections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Sections) == 0 {
		t.Fatal("GET /guideline-sections returned no report categories")
	}
	for _, want := range []string{"community.harassment", "creator.content_ratings"} {
		found := false
		for _, section := range payload.Sections {
			if section.Code == want && section.Label != "" && section.Corpus != "" && section.Sort > 0 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GET /guideline-sections is missing valid section %q", want)
		}
	}
}

func TestReportSubmissionAndHistoryRequireAuthentication(t *testing.T) {
	app := newTestApp()
	requests := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{"target_type":"user","target_id":1,"reason":"community.harassment","description":"This is a sufficiently detailed report."}`)),
		httptest.NewRequest(http.MethodGet, "/me/reports", nil),
	}
	requests[0].Header.Set("Content-Type", "application/json")
	for _, req := range requests {
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want %d", req.Method, req.URL.Path, resp.StatusCode, http.StatusUnauthorized)
		}
	}
}

func TestGroupsListServesJSONContractToFrontend(t *testing.T) {
	for _, accept := range []string{"application/json", "*/*"} {
		t.Run(accept, func(t *testing.T) {
			app := newTestApp()
			req := httptest.NewRequest(http.MethodGet, "/groups?sort=trending&limit=9&offset=0", nil)
			req.Header.Set("Accept", accept)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("GET /groups with Accept %q status = %d, want %d when the database is unavailable", accept, resp.StatusCode, http.StatusServiceUnavailable)
			}
			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["error"] != "Groups service unavailable" {
				t.Fatalf("GET /groups returned misleading placeholder response: %+v", payload)
			}
		})
	}
}

func TestGroupMutationsRequireAuthentication(t *testing.T) {
	app := newTestApp()
	for _, request := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/groups", body: `{"name":"Test group"}`},
		{method: http.MethodPost, path: "/groups/1/join"},
		{method: http.MethodPost, path: "/groups/1/leave"},
	} {
		t.Run(request.path, func(t *testing.T) {
			req := httptest.NewRequest(request.method, request.path, strings.NewReader(request.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d", request.method, request.path, resp.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestSocialMutationsRequireAuthentication(t *testing.T) {
	app := newTestApp()
	for _, request := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/friends/by-username/example/request"},
		{method: http.MethodPost, path: "/friends/send/2"},
		{method: http.MethodPost, path: "/friends/accept/2"},
		{method: http.MethodPost, path: "/friends/decline/2"},
		{method: http.MethodPost, path: "/friends/cancel/2"},
		{method: http.MethodPost, path: "/friends/remove/2"},
		{method: http.MethodPost, path: "/friends/2/block"},
		{method: http.MethodPost, path: "/friends/2/unblock"},
		{method: http.MethodDelete, path: "/friends/2"},
	} {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			req := httptest.NewRequest(request.method, request.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d", request.method, request.path, resp.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestPrivateProfileAPIsRequireAuthentication(t *testing.T) {
	app := newTestApp()
	for _, path := range []string{"/me/groups", "/me/places", "/me/friends", "/me/friends/pending"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestProfileAPIsFailExplicitlyWhenServicesAreUnavailable(t *testing.T) {
	app := newTestApp()
	for _, path := range []string{"/users/1/inventory", "/users/1/places", "/users/1/friends", "/users/1/groups"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestGameJoinReturnsNotImplementedInsteadOfFakeTicket(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodPost, "/places/1/join", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("POST /places/1/join status = %d, want %d", resp.StatusCode, http.StatusNotImplemented)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["code"] != "not_implemented" || payload["error"] == "" {
		t.Fatalf("POST /places/1/join returned a success-shaped or incomplete response: %+v", payload)
	}
	if strings.Contains(string(body), "127.0.0.1") || strings.Contains(string(body), "join_ticket") {
		t.Fatalf("POST /places/1/join returned a fabricated game server ticket: %s", body)
	}
}

func TestGameRatingDoesNotInventVoteCounts(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/places/1/rating", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /places/1/rating status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var payload struct {
		ThumbsUp   int `json:"thumbs_up"`
		ThumbsDown int `json:"thumbs_down"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.ThumbsUp != 0 || payload.ThumbsDown != 0 {
		t.Fatalf("GET /places/1/rating fabricated vote counts: %+v", payload)
	}
}

func TestCatalogItemVariantsReturnFrontendCompatibleList(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source file")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	app := newTestApp()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/items/183/variants", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /items/183/variants status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var payload struct {
		Variants []json.RawMessage `json:"variants"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Variants == nil || len(payload.Variants) != 0 {
		t.Fatalf("GET /items/183/variants returned %#v, want an empty list for a non-variant item", payload.Variants)
	}

	missing, err := app.Test(httptest.NewRequest(http.MethodGet, "/items/999999/variants", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /items/999999/variants status = %d, want %d", missing.StatusCode, http.StatusNotFound)
	}
}

func TestAvatarOutfitAPIsRequireAuthentication(t *testing.T) {
	app := newTestApp()
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/me/outfits"},
		{http.MethodPost, "/me/outfits"},
		{http.MethodPut, "/me/outfits/1"},
		{http.MethodDelete, "/me/outfits/1"},
		{http.MethodPost, "/me/outfits/1/wear"},
		{http.MethodPut, "/me/equipped"},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d", test.method, test.path, resp.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestCreatePageIsRegisteredAndRequiresLogin(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/create?type=game", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET /create status = %d, want login redirect (%d)", resp.StatusCode, http.StatusSeeOther)
	}
	if got := resp.Header.Get("Location"); got != "/login" {
		t.Fatalf("GET /create redirect = %q, want /login", got)
	}
}

func TestSearchPageRedirectsToSearchResults(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/search?q=builder", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET /search status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if got := resp.Header.Get("Location"); got != "/shop?q=builder" {
		t.Fatalf("GET /search redirect = %q, want /shop?q=builder", got)
	}
}

func TestLegacyPagesRedirectToFrontendRoutes(t *testing.T) {
	app := newTestApp()
	for path, destination := range map[string]string{
		"/auth/studio": "https://buildoverse.com/studio",
		"/groups":      "https://buildoverse.com/groups",
		"/trade":       "https://buildoverse.com/trade",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusSeeOther {
				t.Fatalf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusSeeOther)
			}
			if got := resp.Header.Get("Location"); got != destination {
				t.Fatalf("GET %s redirect = %q, want %q", path, got, destination)
			}
		})
	}
}

func TestPlacesDiscoveryMatchesFrontendShelfShape(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/places/discovery?surface=discover&limit=36", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /places/discovery status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var payload struct {
		Surface string `json:"surface"`
		Shelves []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Places  []any  `json:"places"`
			Shelves []any  `json:"shelves"`
		} `json:"shelves"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Surface != "discover" || len(payload.Shelves) != 1 {
		t.Fatalf("unexpected discovery payload: surface=%q shelves=%d", payload.Surface, len(payload.Shelves))
	}
	if payload.Shelves[0].ID == "" || payload.Shelves[0].Title == "" || payload.Shelves[0].Places == nil {
		t.Fatalf("discovery shelf is missing required properties: %+v", payload.Shelves[0])
	}
	if payload.Shelves[0].Shelves != nil {
		t.Fatal("discovery shelf is incorrectly nested inside another shelves property")
	}
}

func TestSPASettingsAndInventoryAPIPathsAreRegistered(t *testing.T) {
	app := newTestApp()
	for _, path := range []string{"/me/settings", "/me/inventory", "/me/ignores", "/me/blocks"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Accept", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("GET %s status = %d, want %d for anonymous request", path, resp.StatusCode, http.StatusUnauthorized)
			}
			if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
				t.Fatalf("GET %s did not return a JSON authentication error", path)
			}
		})
	}
	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/me/settings"},
		{http.MethodPost, "/me/ignores/2"},
		{http.MethodDelete, "/me/ignores/2"},
		{http.MethodPost, "/me/blocks/2"},
		{http.MethodDelete, "/me/blocks/2"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d for anonymous request", route.method, route.path, resp.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestPlacesDiscoveryRejectsUnknownSurface(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/places/discovery?surface=unknown", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /places/discovery with unknown surface status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

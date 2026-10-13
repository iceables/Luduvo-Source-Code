package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

type catalogDoc struct {
	Items []map[string]any `json:"items"`
}

func StorePage(c fiber.Ctx) error {
	if wantsJSON(c) || (c.Get("HX-Request") != "true" && !strings.Contains(c.Get("Accept"), "text/html")) {
		return CatalogStore(c)
	}
	return UpgradePage(c)
}

func loadCatalog() catalogDoc {
	raw, err := os.ReadFile("data/catalog.json")
	if err != nil {
		return catalogDoc{}
	}
	var doc catalogDoc
	_ = json.Unmarshal(raw, &doc)
	return doc
}

func itemKind(item map[string]any) string {
	asset, _ := item["asset_type"].(string)
	slot, _ := item["slot"].(string)
	switch asset {
	case "flat_shirt", "flat_pant":
		return ""
	case "face":
		return "face"
	case "avatar_bundle":
		return "avatar_bundle"
	case "accessory":
		if slot == "hat" {
			return "hat"
		}
	}
	return ""
}

func CatalogStore(c fiber.Ctx) error {
	doc := loadCatalog()
	want := strings.ToLower(c.Query("asset_type"))
	if want == "accessory" {
		want = "hat"
	}
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit < 1 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	matched := make([]map[string]any, 0)
	for _, item := range doc.Items {
		kind := itemKind(item)
		if kind == "" {
			continue
		}
		if want != "" && want != kind && !(want == "hat" && kind == "hat") {
			continue
		}
		if q != "" {
			name, _ := item["name"].(string)
			desc, _ := item["description"].(string)
			if !strings.Contains(strings.ToLower(name+" "+desc), q) {
				continue
			}
		}
		matched = append(matched, catalogItemResponse(item))
	}
	if offset > len(matched) {
		offset = len(matched)
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	return c.JSON(fiber.Map{
		"items":  matched[offset:end],
		"total":  len(matched),
		"limit":  limit,
		"offset": offset,
	})
}

func CatalogItem(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	item, ok := findCatalogItem(id)
	if ok {
		return c.JSON(catalogItemResponse(item))
	}
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}

func catalogItemResponse(item map[string]any) map[string]any {
	response := make(map[string]any, len(item))
	for key, value := range item {
		response[key] = value
	}
	rebrandCatalogCreator(response)

	modelURL, ok := item["model_url"].(string)
	if !ok {
		return response
	}
	parsed, err := url.Parse(modelURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "assets.luduvo.com" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(!strings.HasPrefix(parsed.Path, "/items/assets/") && !strings.HasPrefix(parsed.Path, "/scenes/")) {
		return response
	}
	cleanPath := filepath.ToSlash(filepath.Clean(parsed.Path))
	if cleanPath != parsed.Path || strings.Contains(parsed.Path, "\\") {
		return response
	}
	ext := strings.ToLower(filepath.Ext(parsed.Path))
	if ext != ".png" && ext != ".glb" && ext != ".ldv" {
		return response
	}
	response["model_url"] = "https://api.buildoverse.com/model-asset?path=" + url.QueryEscape(parsed.Path)
	return response
}

func CatalogModelAsset(c fiber.Ctx) error {
	assetPath := c.Query("path")
	cleanPath := filepath.ToSlash(filepath.Clean(assetPath))
	if assetPath == "" || cleanPath != assetPath || strings.Contains(assetPath, "\\") ||
		(!strings.HasPrefix(assetPath, "/items/assets/") && !strings.HasPrefix(assetPath, "/scenes/")) {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	ext := strings.ToLower(filepath.Ext(assetPath))
	contentType := map[string]string{
		".png": "image/png",
		".glb": "model/gltf-binary",
		".ldv": "application/octet-stream",
	}[ext]
	if contentType == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://assets.luduvo.com"+assetPath, nil)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			if request.URL.Scheme != "https" || request.URL.Hostname() != "assets.luduvo.com" {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Model asset could not be fetched"})
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Model asset upstream returned an error"})
	}

	const maxAssetSize = 64 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAssetSize+1))
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Model asset could not be read"})
	}
	if len(body) > maxAssetSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "Model asset is too large"})
	}

	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=86400, immutable")
	return c.Send(body)
}

func ItemsVariants(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	item, ok := findCatalogItem(id)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	variants, ok := item["variants"].([]any)
	if !ok {
		if accessory, isAccessory := item["accessory"].(map[string]any); isAccessory {
			variants, ok = accessory["variants"].([]any)
		}
	}
	if !ok || variants == nil {
		variants = []any{}
	}
	return c.JSON(fiber.Map{"variants": variants})
}

func findCatalogItem(id int) (map[string]any, bool) {
	for _, item := range loadCatalog().Items {
		if itemID, ok := item["id"].(float64); ok && int(itemID) == id && itemKind(item) != "" {
			return item, true
		}
	}
	return nil, false
}

func CatalogThumbnail(c fiber.Ctx) error {
	id, err := strconv.Atoi(strings.TrimSuffix(c.Params("id"), ".img"))
	if err != nil || id <= 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}

	path := filepath.Join("static", "catalog", strconv.Itoa(id)+".img")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return err
	}

	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=86400")
	c.Set("Access-Control-Allow-Origin", "*")
	return c.SendFile(path)
}

func rebrandCatalogCreator(response map[string]any) {
	for _, key := range []string{"creator_username", "creator_name"} {
		name, ok := response[key].(string)
		if ok && strings.EqualFold(strings.TrimSpace(name), "Luduvo") {
			response[key] = "Buildoverse"
		}
	}
}

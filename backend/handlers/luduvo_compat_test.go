package handlers

import (
	"database/sql"
	"fmt"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestPlaceTagMapsAlwaysReturnsArray(t *testing.T) {
	tests := []struct {
		name string
		raw  sql.NullString
		want int
	}{
		{name: "missing tags", want: 0},
		{name: "json names", raw: sql.NullString{String: `["Adventure","Puzzle"]`, Valid: true}, want: 2},
		{name: "json objects", raw: sql.NullString{String: `[{"name":"Adventure"}]`, Valid: true}, want: 1},
		{name: "comma separated", raw: sql.NullString{String: "Adventure, Puzzle", Valid: true}, want: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := placeTagMaps(test.raw)
			if got == nil {
				t.Fatal("placeTagMaps returned nil")
			}
			if len(got) != test.want {
				t.Fatalf("placeTagMaps returned %d tags, want %d", len(got), test.want)
			}
		})
	}
}

func TestCatalogItemResponseProxiesApprovedModelAssets(t *testing.T) {
	item := map[string]any{
		"id":        183,
		"model_url": "https://assets.luduvo.com/scenes/0123456789abcdef.ldv",
	}

	got := catalogItemResponse(item)
	proxied, ok := got["model_url"].(string)
	if !ok {
		t.Fatalf("model_url has type %T, want string", got["model_url"])
	}
	parsed, err := url.Parse(proxied)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Hostname() != "api.buildoverse.com" || parsed.Path != "/model-asset" ||
		parsed.Query().Get("path") != "/scenes/0123456789abcdef.ldv" {
		t.Fatalf("model_url = %q, not the expected same-origin proxy URL", proxied)
	}
	if item["model_url"] != "https://assets.luduvo.com/scenes/0123456789abcdef.ldv" {
		t.Fatal("catalog item was mutated while building its response")
	}
}

func TestCatalogItemResponseCreditsBuildoverseInsteadOfLuduvo(t *testing.T) {
	got := catalogItemResponse(map[string]any{
		"creator_username": "Luduvo",
		"creator_name":     "luduvo",
		"name":             "Pasta Strainer",
	})
	if got["creator_username"] != "Buildoverse" || got["creator_name"] != "Buildoverse" {
		t.Fatalf("creator credit = (%v, %v), want Buildoverse", got["creator_username"], got["creator_name"])
	}
	if got["name"] != "Pasta Strainer" {
		t.Fatal("item name was rewritten")
	}
}

func TestCatalogItemResponseDoesNotProxyUnapprovedModelHosts(t *testing.T) {
	modelURL := "https://attacker.example/items/assets/model.glb"
	got := catalogItemResponse(map[string]any{"model_url": modelURL})
	if got["model_url"] != modelURL {
		t.Fatalf("unapproved model_url was rewritten to %v", got["model_url"])
	}
}

func TestDefaultAvatarRulesUseExistingCatalogItemsAndBuildoversePalette(t *testing.T) {
	rules := defaultAvatarRules()
	if fmt.Sprint(rules["default_bundle_item_id"]) != "17" ||
		fmt.Sprint(rules["default_face_item_id"]) != "19" {
		t.Fatalf("default item IDs = (%v, %v), want (17, 19)",
			rules["default_bundle_item_id"], rules["default_face_item_id"])
	}

	var colors map[string]any
	switch value := rules["default_body_colors"].(type) {
	case map[string]any:
		colors = value
	case fiber.Map:
		colors = map[string]any(value)
	}
	if colors == nil {
		t.Fatalf("default_body_colors has type %T", rules["default_body_colors"])
	}
	for part, want := range map[string]string{
		"head": "#FFFFFF", "left_arm": "#FFFFFF", "right_arm": "#FFFFFF",
		"left_leg": "#FF0000", "right_leg": "#FF0000", "torso": "#8B0000",
	} {
		if got, _ := colors[part].(string); got != want {
			t.Errorf("default_body_colors[%q] = %q, want %q", part, got, want)
		}
	}
}

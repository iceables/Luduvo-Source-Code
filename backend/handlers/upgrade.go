package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"buildoverse/backend/config"
)

type lunePack struct {
	SKU    string
	Lunes  int
	Price  string
	Bonus  int
	Image  string
}

type planTier struct {
	ID           string
	Name         string
	Monthly      string
	Yearly       string
	MonthlyCents int
	YearlyCents  int
	Blurb        string
	Includes     string
	Perks        []string
	Preview      string
	Cosmetic     string
}

type subRecord struct {
	Username string `json:"username"`
	Tier     string `json:"tier"`
	Interval string `json:"interval"`
	SKU      string `json:"sku"`
	Lunes    int    `json:"lunes"`
	Updated  string `json:"updated"`
}

var (
	subMu   sync.Mutex
	subFile = filepath.Join("data", "subscriptions.json")
)

func storePacks() []lunePack {
	return []lunePack{
		{SKU: "lunes_100", Lunes: 100, Price: "$0.99", Image: "1"},
		{SKU: "lunes_550", Lunes: 550, Price: "$4.99", Bonus: 50, Image: "2"},
		{SKU: "lunes_1100", Lunes: 1100, Price: "$9.99", Bonus: 100, Image: "3"},
		{SKU: "lunes_2200", Lunes: 2200, Price: "$19.99", Bonus: 200, Image: "4"},
		{SKU: "lunes_5500", Lunes: 5500, Price: "$49.99", Bonus: 500, Image: "5"},
		{SKU: "lunes_12000", Lunes: 12000, Price: "$99.99", Bonus: 2000, Image: "6"},
		{SKU: "lunes_25500", Lunes: 25500, Price: "$199.99", Bonus: 5500, Image: "7"},
	}
}

func storePlans() []planTier {
	return []planTier{
		{
			ID: "cadet", Name: "Cadet", Monthly: "$4.99", Yearly: "$49.99",
			MonthlyCents: 499, YearlyCents: 4999,
			Blurb: "Daily Lunes and the Cadet look.",
			Perks: []string{"25 Lunes every day", "Exclusive Cadet cosmetic", "250 friends", "125 groups", "250 trades a day", "Advanced profile", "Free username changes", "Free flat clothing uploads", "Custom themes"},
			Preview: "/static/images/store/cadet-preview.webp", Cosmetic: "/static/images/store/cadet1.gif",
		},
		{
			ID: "specialist", Name: "Specialist", Monthly: "$9.99", Yearly: "$99.99",
			MonthlyCents: 999, YearlyCents: 9999,
			Includes: "Everything in Cadet, and:",
			Perks:    []string{"60 Lunes every day", "Exclusive Specialist cosmetic", "Custom profile picture", "400 friends", "200 groups"},
			Preview:  "/static/images/store/specialist-preview.webp", Cosmetic: "/static/images/store/specialist1.gif",
		},
		{
			ID: "commander", Name: "Commander", Monthly: "$19.99", Yearly: "$199.99",
			MonthlyCents: 1999, YearlyCents: 19999,
			Includes: "Everything in Specialist, and:",
			Perks:    []string{"150 Lunes every day", "Exclusive Commander cosmetic", "800 friends", "400 groups"},
			Preview:  "/static/images/store/commander-preview.webp", Cosmetic: "/static/images/store/commander1.gif",
		},
	}
}

func loadSubs() map[string]subRecord {
	out := map[string]subRecord{}
	b, err := os.ReadFile(subFile)
	if err != nil || len(b) == 0 {
		return out
	}
	_ = json.Unmarshal(b, &out)
	return out
}

func saveSub(rec subRecord) {
	subMu.Lock()
	defer subMu.Unlock()
	all := loadSubs()
	all[strings.ToLower(rec.Username)] = rec
	_ = os.MkdirAll(filepath.Dir(subFile), 0o755)
	b, _ := json.MarshalIndent(all, "", "  ")
	_ = os.WriteFile(subFile, b, 0o644)
}

func currentPlan(username string) string {
	if username == "" {
		return ""
	}
	subMu.Lock()
	defer subMu.Unlock()
	rec, ok := loadSubs()[strings.ToLower(username)]
	if !ok {
		return ""
	}
	return rec.Tier
}

func UpgradePage(c fiber.Ctx) error {
	username := GetActiveUser(c)
	interval := c.Query("interval", "monthly")
	if interval != "yearly" {
		interval = "monthly"
	}
	msg := ""
	if c.Query("checkout") == "return" {
		msg = "Payment landed. Your plan and Lunes are on the account."
	}
	if c.Query("checkout") == "cancel" {
		msg = "Checkout closed. Nothing was charged."
	}
	return Render(c, "pages/upgrade", fiber.Map{
		"Title":    "Store - Buildoverse",
		"Packs":    storePacks(),
		"Plans":    storePlans(),
		"Interval": interval,
		"Plan":     currentPlan(username),
		"Message":  msg,
		"LoggedIn": username != "",
	}, "layouts/main")
}

func UpgradeCheckout(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sign in before you buy."})
	}
	sku := strings.TrimSpace(c.FormValue("sku"))
	if sku == "" {
		sku = strings.TrimSpace(c.Query("sku"))
	}
	interval := c.FormValue("interval")
	if interval == "" {
		interval = "monthly"
	}

	kind, name, cents, lunes, tier := resolveSKU(sku, interval)
	if kind == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Unknown product."})
	}

	origin := c.Protocol() + "://" + c.Hostname()
	successURL := origin + "/upgrade?checkout=return&sku=" + url.QueryEscape(sku)
	cancelURL := origin + "/upgrade?checkout=cancel"

	secret := os.Getenv("STRIPE_SECRET_KEY")
	if secret == "" && config.Global != nil {
		secret = os.Getenv("STRIPE_SECRET")
	}
	if secret != "" {
		sessURL, err := stripeCheckout(secret, username, sku, name, cents, successURL, cancelURL, interval)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
		}
		if c.Get("HX-Request") == "true" {
			c.Set("HX-Redirect", sessURL)
			return c.SendStatus(fiber.StatusSeeOther)
		}
		return c.Redirect().To(sessURL)
	}

	rec := subRecord{
		Username: username,
		SKU:      sku,
		Interval: interval,
		Lunes:    lunes,
		Updated:  time.Now().UTC().Format(time.RFC3339),
	}
	if kind == "subscription" {
		rec.Tier = tier
	} else if prev := currentPlan(username); prev != "" {
		rec.Tier = prev
	}
	saveSub(rec)
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", successURL)
		return c.SendStatus(fiber.StatusSeeOther)
	}
	return c.Redirect().To(successURL)
}

func resolveSKU(sku, interval string) (kind, name string, cents, lunes int, tier string) {
	for _, p := range storePacks() {
		if p.SKU == sku {
			cents = parseCents(p.Price)
			return "lune_pack", fmt.Sprintf("%d Lunes", p.Lunes+p.Bonus), cents, p.Lunes + p.Bonus, ""
		}
	}
	for _, p := range storePlans() {
		if sku == p.ID+"_monthly" || (sku == p.ID && interval != "yearly") {
			return "subscription", p.Name + " monthly", p.MonthlyCents, 0, p.ID
		}
		if sku == p.ID+"_yearly" || (sku == p.ID && interval == "yearly") {
			return "subscription", p.Name + " yearly", p.YearlyCents, 0, p.ID
		}
	}
	return "", "", 0, 0, ""
}

func parseCents(price string) int {
	price = strings.TrimPrefix(price, "$")
	var dollars, cents int
	fmt.Sscanf(price, "%d.%d", &dollars, &cents)
	return dollars*100 + cents
}

func stripeCheckout(secret, username, sku, name string, cents int, successURL, cancelURL, interval string) (string, error) {
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	form.Set("client_reference_id", username)
	form.Set("metadata[sku]", sku)
	form.Set("metadata[interval]", interval)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", "usd")
	form.Set("line_items[0][price_data][unit_amount]", fmt.Sprintf("%d", cents))
	form.Set("line_items[0][price_data][product_data][name]", name)
	req, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(secret, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("stripe %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.URL == "" {
		return "", fmt.Errorf("stripe returned no checkout url")
	}
	return parsed.URL, nil
}

func setSignupFlash(c fiber.Ctx, username string) {
	sess := session.FromContext(c)
	if sess == nil {
		return
	}
	sess.Set("signup_message", fmt.Sprintf("Welcome to Buildoverse, %s. Your avatar is waiting in the editor — paint the body, wear what you own, and the place list is already open.", username))
}

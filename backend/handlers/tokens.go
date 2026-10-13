package handlers

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

)

type issuedToken struct {
	UserID int    `json:"user_id"`
	Kind   string `json:"kind"`
}

var (
	tokenMu    sync.Mutex
	tokenCache map[string]issuedToken
)

func tokenPath() string { return "data/tokens.json" }

func loadTokens() map[string]issuedToken {
	if tokenCache != nil {
		return tokenCache
	}
	tokenCache = map[string]issuedToken{}
	b, err := os.ReadFile(tokenPath())
	if err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &tokenCache)
	}
	if tokenCache == nil {
		tokenCache = map[string]issuedToken{}
	}
	return tokenCache
}

func saveTokens() {
	_ = os.MkdirAll("data", 0o755)
	b, _ := json.Marshal(loadTokens())
	_ = os.WriteFile(tokenPath(), b, 0o600)
}

func rememberToken(token string, userID int, kind string) {
	if token == "" || userID == 0 {
		return
	}
	tokenMu.Lock()
	defer tokenMu.Unlock()
	loadTokens()[token] = issuedToken{UserID: userID, Kind: kind}
	saveTokens()
}

func lookupToken(token string) (issuedToken, bool) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	rec, ok := loadTokens()[token]
	return rec, ok
}

func userIDFromAuthHeader(header string) int {
	token := strings.TrimSpace(header)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	rec, ok := lookupToken(token)
	if !ok {
		return 0
	}
	return rec.UserID
}

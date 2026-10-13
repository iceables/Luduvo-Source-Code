package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

const (
	PowerMember          = 0
	PowerTrainee         = 1
	PowerModerator       = 2
	PowerHeadManager     = 3
	PowerBuildoverseTeam = 4
	PowerOwner           = 5
)

type User struct {
	ID                int
	Username          string
	DisplayName       string
	Mail              string
	Password          string
	Description       string
	Unikey            string
	Power             int
	PrimaryClan       int
	NameColor         string
	CustomCSS         sql.NullString
	Vermail           string
	Vermc             string
	Casom             string
	Bits              int
	Bucks             int
	LastOnline        time.Time
	CreationDate      time.Time
	Views             int
	FeedCaptchas      int
	EmailVerifyToken  sql.NullString
	EmailVerifyExpiry sql.NullInt64
	Pronouns          string
	Socials           string
	MusicID           sql.NullInt64
}

type UsernameHistory struct {
	Username  string `json:"username"`
	ClaimedAt int64  `json:"claimed_at"`
	Current   bool   `json:"current"`
}

func (u *User) RoleName() string {
	switch {
	case u.Power >= PowerOwner:
		return "owner"
	case u.Power == PowerBuildoverseTeam:
		return "buildoverse-team"
	case u.Power == PowerHeadManager:
		return "head manager"
	case u.Power == PowerModerator:
		return "moderator"
	case u.Power == PowerTrainee:
		return "trainee"
	default:
		return "member"
	}
}

func (u *User) RoleDisplayName() string {
	switch {
	case u.Power >= PowerOwner:
		return "Owner"
	case u.Power == PowerBuildoverseTeam:
		return "Buildoverse Team"
	case u.Power == PowerHeadManager:
		return "Head Manager"
	case u.Power == PowerModerator:
		return "Moderator"
	case u.Power == PowerTrainee:
		return "Trainee"
	default:
		return "Member"
	}
}

func (u *User) HasAdminAccess() bool {
	return u.Power >= PowerTrainee
}

func (u *User) HasPower(required int) bool {
	return u.Power >= required
}

type SocialLink struct {
	Platform string `json:"platform"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	URL      string `json:"url"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}

func (u *User) ParsedSocials() []SocialLink {
	if u.Socials == "" {
		return nil
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(u.Socials), &raw); err != nil {
		return nil
	}

	platforms := []struct {
		Key    string
		Name   string
		Icon   string
		Color  string
		URLFmt func(v string) string
	}{
		{"discord", "Discord", "fa-brands fa-discord", "#5865F2", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			return "https://discord.com/users/" + v
		}},
		{"twitter", "Twitter / X", "fa-brands fa-x-twitter", "#000000", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			return "https://x.com/" + strings.TrimPrefix(v, "@")
		}},
		{"youtube", "YouTube", "fa-brands fa-youtube", "#FF0000", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			if !strings.HasPrefix(v, "@") && !strings.HasPrefix(v, "c/") && !strings.HasPrefix(v, "channel/") {
				return "https://youtube.com/@" + v
			}
			return "https://youtube.com/" + v
		}},
		{"twitch", "Twitch", "fa-brands fa-twitch", "#9146FF", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			return "https://twitch.tv/" + v
		}},
		{"github", "GitHub", "fa-brands fa-github", "#24292e", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			return "https://github.com/" + v
		}},
		{"instagram", "Instagram", "fa-brands fa-instagram", "#E1306C", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			return "https://instagram.com/" + strings.TrimPrefix(v, "@")
		}},
		{"tiktok", "TikTok", "fa-brands fa-tiktok", "#000000", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			return "https://tiktok.com/@" + strings.TrimPrefix(v, "@")
		}},
		{"steam", "Steam", "fa-brands fa-steam", "#171a21", func(v string) string {
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				return v
			}
			return "https://steamcommunity.com/id/" + v
		}},
	}

	var result []SocialLink
	for _, p := range platforms {
		if val, ok := raw[p.Key]; ok {
			trimmed := strings.TrimSpace(val)
			if trimmed != "" {
				result = append(result, SocialLink{
					Platform: p.Key,
					Name:     p.Name,
					Value:    trimmed,
					URL:      p.URLFmt(trimmed),
					Icon:     p.Icon,
					Color:    p.Color,
				})
			}
		}
	}
	return result
}

type ReplayCache struct {
	mu     sync.RWMutex
	nonces map[string]time.Time
}

func NewReplayCache() *ReplayCache {
	rc := &ReplayCache{
		nonces: make(map[string]time.Time),
	}
	go rc.cleanupLoop()
	return rc
}

func (rc *ReplayCache) Add(nonce string, expiry time.Time) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if _, exists := rc.nonces[nonce]; exists {
		return false
	}
	rc.nonces[nonce] = expiry
	return true
}

func (rc *ReplayCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for nonce, expiry := range rc.nonces {
			if now.After(expiry) {
				delete(rc.nonces, nonce)
			}
		}
		rc.mu.Unlock()
	}
}

func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"buildoverse/backend/models"
)

type MusicService struct {
	client *http.Client
}

func NewMusicService() *MusicService {
	return &MusicService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

var tagBracketRegex = regexp.MustCompile(`(?i)\b(artist|track|album|label):\s*\[([^\]]+)\]`)

const deezerUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func (s *MusicService) getJSON(apiURL string, dst any) error {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", deezerUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deezer api returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(dst)
}

func (s *MusicService) SearchTracks(query string) ([]models.DeezerTrack, error) {
	if query == "" {
		return []models.DeezerTrack{}, nil
	}

	formattedQuery := tagBracketRegex.ReplaceAllStringFunc(query, func(m string) string {
		match := tagBracketRegex.FindStringSubmatch(m)
		if len(match) == 3 {
			tagName := strings.ToLower(match[1])
			val := strings.TrimSpace(match[2])
			return fmt.Sprintf(`%s:"%s"`, tagName, val)
		}
		return m
	})

	apiURL := fmt.Sprintf("https://api.deezer.com/search?q=%s", url.QueryEscape(formattedQuery))

	var deezerResp models.DeezerSearchResponse
	if err := s.getJSON(apiURL, &deezerResp); err != nil {
		return nil, err
	}

	for i := range deezerResp.Data {
		mins := deezerResp.Data[i].Duration / 60
		secs := deezerResp.Data[i].Duration % 60
		deezerResp.Data[i].Formatted = fmt.Sprintf("%d:%02d", mins, secs)
	}

	return deezerResp.Data, nil
}

func (s *MusicService) GetTrackByID(trackID int64) (*models.DeezerTrack, error) {
	if trackID <= 0 {
		return nil, errors.New("invalid track id")
	}

	apiURL := fmt.Sprintf("https://api.deezer.com/track/%d", trackID)

	var track models.DeezerTrack
	if err := s.getJSON(apiURL, &track); err != nil {
		return nil, err
	}

	if track.ID == 0 || track.Title == "" {
		return nil, errors.New("track not found")
	}

	mins := track.Duration / 60
	secs := track.Duration % 60
	track.Formatted = fmt.Sprintf("%d:%02d", mins, secs)

	return &track, nil
}
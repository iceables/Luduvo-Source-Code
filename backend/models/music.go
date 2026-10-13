package models

type DeezerArtist struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	PictureSmall  string `json:"picture_small"`
	PictureMedium string `json:"picture_medium"`
}

type DeezerAlbum struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	CoverSmall  string `json:"cover_small"`
	CoverMedium string `json:"cover_medium"`
}

type DeezerTrack struct {
	ID        int64        `json:"id"`
	Title     string       `json:"title"`
	Duration  int          `json:"duration"`
	Preview   string       `json:"preview"`
	Artist    DeezerArtist `json:"artist"`
	Album     DeezerAlbum  `json:"album"`
	Formatted string       `json:"formatted_duration,omitempty"`
}

type DeezerSearchResponse struct {
	Data  []DeezerTrack `json:"data"`
	Total int           `json:"total"`
}
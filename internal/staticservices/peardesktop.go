package staticservices

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
)

// SongInfo represents detailed song information from Pear Desktop
type SongInfo struct {
	Title          string  `json:"title"`
	Artist         string  `json:"artist"`
	Views          int     `json:"views"`
	SongDuration   int     `json:"songDuration"`
	ElapsedSeconds int     `json:"elapsedSeconds,omitempty"`
	VideoID        string  `json:"videoId"`
	PlaylistID     string  `json:"playlistId"`
	ImageSrc       *string `json:"imageSrc,omitempty"`
	Album          *string `json:"album,omitempty"`
	URL            string  `json:"url"`
	MediaType      string  `json:"mediaType"`
	IsPaused       bool    `json:"isPaused,omitempty"`
}

// VolumeState represents the volume state
type VolumeState struct {
	State   int  `json:"state"`
	IsMuted bool `json:"isMuted"`
}

// MusicPlayerState represents the current state of the music player (simplified for our API)
type MusicPlayerState struct {
	IsPlaying      bool   `json:"isPlaying"`
	CurrentSong    string `json:"currentSong,omitempty"`
	Artist         string `json:"artist,omitempty"`
	URL            string `json:"url,omitempty"`
	SongDuration   int    `json:"songDuration,omitempty"`
	ImageSrc       string `json:"imageSrc,omitempty"`
	ElapsedSeconds int    `json:"elapsedSeconds,omitempty"`
	Volume         int    `json:"volume,omitempty"`
}

// PearDesktopService handles communication with the Pear Desktop background process
type PearDesktopService struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

// NewPearDesktopService creates a new Pear Desktop service instance
func NewPearDesktopService() *PearDesktopService {
	return &PearDesktopService{
		baseURL:   "http://" + data.GetPearDesktopHost(),
		authToken: "", // Empty token as mentioned by user
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// makeRequest creates an HTTP request with proper authentication
func (s *PearDesktopService) makeRequest(method, endpoint string, body interface{}) (*http.Request, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := s.baseURL + endpoint
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, reqBody)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	// Add Bearer token authentication
	req.Header.Set("Authorization", "Bearer "+s.authToken)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// TestConnection verifies the connection to the Pear Desktop background process
func (s *PearDesktopService) TestConnection() error {
	req, err := s.makeRequest("GET", "/api/v1/song", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Pear Desktop: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Pear Desktop health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetInitialPlayerState fetches the complete initial player state from Pear Desktop
func (s *PearDesktopService) GetInitialPlayerState() (*MusicPlayerState, error) {
	// Get current song information
	song, err := s.GetCurrentSong()
	if err != nil {
		return nil, fmt.Errorf("failed to get current song: %w", err)
	}

	// Get volume information
	volume, err := s.GetVolume()
	if err != nil {
		return nil, fmt.Errorf("failed to get volume: %w", err)
	}

	state := &MusicPlayerState{
		Volume: volume.State,
	}

	// If there's a current song, populate the song information
	if song != nil {
		state.IsPlaying = !song.IsPaused
		state.CurrentSong = song.Title
		state.Artist = song.Artist
		state.URL = song.URL
		state.SongDuration = song.SongDuration
		if song.ImageSrc != nil {
			state.ImageSrc = *song.ImageSrc
		}
		state.ElapsedSeconds = song.ElapsedSeconds
	}

	return state, nil
}

// GetCurrentSong retrieves detailed current song information
func (s *PearDesktopService) GetCurrentSong() (*SongInfo, error) {
	req, err := s.makeRequest("GET", "/api/v1/song", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get current song: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // No song playing
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get current song, status: %d", resp.StatusCode)
	}

	var song SongInfo
	if err := json.NewDecoder(resp.Body).Decode(&song); err != nil {
		return nil, fmt.Errorf("failed to decode song info: %w", err)
	}

	return &song, nil
}

// GetVolume retrieves the current volume state
func (s *PearDesktopService) GetVolume() (*VolumeState, error) {
	req, err := s.makeRequest("GET", "/api/v1/volume", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get volume, status: %d", resp.StatusCode)
	}

	var volume VolumeState
	if err := json.NewDecoder(resp.Body).Decode(&volume); err != nil {
		return nil, fmt.Errorf("failed to decode volume: %w", err)
	}

	return &volume, nil
}

// Play starts playback
func (s *PearDesktopService) Play() error {
	req, err := s.makeRequest("POST", "/api/v1/play", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to play: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to play, status: %d", resp.StatusCode)
	}

	return nil
}

// Pause pauses playback
func (s *PearDesktopService) Pause() error {
	req, err := s.makeRequest("POST", "/api/v1/pause", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pause: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to pause, status: %d", resp.StatusCode)
	}

	return nil
}

// TogglePlay toggles between play and pause
func (s *PearDesktopService) TogglePlay() error {
	req, err := s.makeRequest("POST", "/api/v1/toggle-play", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to toggle play: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to toggle play, status: %d", resp.StatusCode)
	}

	return nil
}

// SetVolume sets the player volume
func (s *PearDesktopService) SetVolume(volume int) error {
	req, err := s.makeRequest("POST", "/api/v1/volume", map[string]int{"volume": volume})
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set volume: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to set volume, status: %d", resp.StatusCode)
	}

	return nil
}

// GetMusicPlayerState retrieves a simplified music player state for backward compatibility
func (s *PearDesktopService) GetMusicPlayerState() (*MusicPlayerState, error) {
	song, err := s.GetCurrentSong()
	if err != nil {
		return nil, err
	}

	volume, err := s.GetVolume()
	if err != nil {
		return nil, err
	}

	currentSong := ""
	if song != nil {
		currentSong = song.Title
	}

	state := &MusicPlayerState{
		IsPlaying:   song != nil && !song.IsPaused,
		CurrentSong: currentSong,
		Volume:      volume.State,
	}

	return state, nil
}

// SetMusicPlayerState provides backward compatibility for setting player state
func (s *PearDesktopService) SetMusicPlayerState(state *MusicPlayerState) error {
	if state.IsPlaying {
		return s.Play()
	} else {
		return s.Pause()
	}
}

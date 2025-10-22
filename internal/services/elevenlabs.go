package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"log"
)

// ElevenLabsTTS synthesizes speech using ElevenLabs API.
// lang should be a normalized tag like yo-NG, ig-NG, ha-NG, but we only use the base language for voice selection.
func ElevenLabsTTS(text, lang string) ([]byte, string, error) {
	apiKey := strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY"))
	if apiKey == "" {
		return nil, "", fmt.Errorf("ELEVENLABS_API_KEY is not configured")
	}

	modelID := strings.TrimSpace(os.Getenv("ELEVENLABS_MODEL_ID"))
	if modelID == "" {
		modelID = "eleven_flash_v2_5"
	}

	voiceDefault := strings.TrimSpace(os.Getenv("ELEVENLABS_VOICE_ID_DEFAULT"))
	voiceYO := strings.TrimSpace(os.Getenv("ELEVENLABS_VOICE_ID_YO"))
	voiceIG := strings.TrimSpace(os.Getenv("ELEVENLABS_VOICE_ID_IG"))
	voiceHA := strings.TrimSpace(os.Getenv("ELEVENLABS_VOICE_ID_HA"))

	l := strings.ToLower(strings.TrimSpace(lang))
	if idx := strings.IndexByte(l, '-'); idx > 0 {
		l = l[:idx]
	}

	var voiceID string
	switch l {
	case "yo":
		voiceID = voiceYO
	case "ig":
		voiceID = voiceIG
	case "ha":
		voiceID = voiceHA
	}
	if voiceID == "" {
		voiceID = voiceDefault
	}
	if voiceID == "" {
		return nil, "", fmt.Errorf("no ElevenLabs voice configured for language: %s", lang)
	}

	payload := map[string]any{
		"text":     text,
		"model_id": modelID,
		// You can adjust voice settings if desired (stability/similarity)
		// "voice_settings": map[string]any{"stability": 0.5, "similarity_boost": 0.75},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("marshal request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}

	// inner function to invoke ElevenLabs
	call := func(vid string) (int, []byte, string, error) {
		u := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", vid)
		req, err := http.NewRequest("POST", u, bytes.NewReader(body))
		if err != nil {
			return 0, nil, "", fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("xi-api-key", apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "audio/mpeg")
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("ElevenLabs request error: %v", err)
			return 0, nil, "", fmt.Errorf("call ElevenLabs: %w", err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		ct := resp.Header.Get("Content-Type")
		return resp.StatusCode, b, ct, nil
	}

	status, respBody, ct, err := call(voiceID)
	if err == nil && status >= 200 && status < 300 {
		if ct == "" { ct = "audio/mpeg" }
		return respBody, ct, nil
	}

	if err == nil && (status == 404 || status == 422 || status == 400) && voiceDefault != "" && voiceID != voiceDefault {
		log.Printf("ElevenLabs: retrying with default voice due to status=%d for voice=%s", status, voiceID)
		status2, body2, ct2, err2 := call(voiceDefault)
		if err2 == nil && status2 >= 200 && status2 < 300 {
			if ct2 == "" { ct2 = "audio/mpeg" }
			return body2, ct2, nil
		}
		if err2 == nil {
			preview := string(body2)
			if len(preview) > 500 { preview = preview[:500] + "..." }
			log.Printf("ElevenLabs default voice error: status=%d body=%s", status2, preview)
			return nil, "", fmt.Errorf("elevenlabs %d: %s", status2, preview)
		}
		return nil, "", err2
	}

	if err != nil {
		return nil, "", err
	}
	preview := string(respBody)
	if len(preview) > 500 { preview = preview[:500] + "..." }
	log.Printf("ElevenLabs error: status=%d body=%s", status, preview)
	return nil, "", fmt.Errorf("elevenlabs %d: %s", status, preview)
}

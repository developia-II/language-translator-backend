package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// SynthesizeTTS uses Hugging Face MMS-TTS models for Yoruba, Igbo, and Hausa.
func SynthesizeTTS(text, lang string) ([]byte, string, error) {
	token := os.Getenv("HF_API_TOKEN")
	if strings.TrimSpace(token) == "" {
		return nil, "", fmt.Errorf("HF_API_TOKEN is not configured")
	}

	// Pick model per language
	var model string
	origLang := strings.ToLower(lang)
	switch strings.ToLower(lang) {
	case "yo", "yo-ng", "yor", "yoruba":
		if v := strings.TrimSpace(os.Getenv("TTS_YOR_MODEL")); v != "" {
			model = v
		} else {
			model = "Xenova/mms-tts-yor"
		}
	case "ig", "ig-ng", "ibo", "igbo":
		if v := strings.TrimSpace(os.Getenv("TTS_IGB_MODEL")); v != "" {
			model = v
		} else {
			model = "facebook/mms-tts-ibo"
		}
	case "ha", "ha-ng", "hau", "hausa":
		if v := strings.TrimSpace(os.Getenv("TTS_HAU_MODEL")); v != "" {
			model = v
		} else {
			model = "facebook/mms-tts-hau"
		}
	default:
		return nil, "", fmt.Errorf("unsupported language for Hugging Face TTS: %s", lang)
	}

	log.Printf("TTS: lang=%s model=%s", origLang, model)

	apiURL := fmt.Sprintf("https://api-inference.huggingface.co/models/%s", model)
	payload := map[string]any{
		"inputs":  text,
		"options": map[string]any{"wait_for_model": true},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("marshal request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}

	var lastStatus int
	var lastBody []byte
	var lastCT string
	for attempt := 0; attempt < 3; attempt++ {
		// Build a fresh request each attempt so the Body is readable every time
		req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
		if err != nil {
			return nil, "", fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "audio/wav")
		req.Header.Set("User-Agent", "language-translator-backend/tts (+github.com/developia-II)")

		resp, err := client.Do(req)
		if err != nil {
			if attempt == 2 {
				return nil, "", fmt.Errorf("call Hugging Face: %w", err)
			}
			log.Printf("TTS request error (attempt %d): %v", attempt+1, err)
			time.Sleep(time.Duration(1+attempt) * 2 * time.Second)
			continue
		}
		func() {
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			lastStatus = resp.StatusCode
			lastBody = b
			lastCT = resp.Header.Get("Content-Type")
		}()
		log.Printf("TTS HF response: status=%d ct=%s len=%d (attempt %d)", lastStatus, lastCT, len(lastBody), attempt+1)

		if lastStatus >= 200 && lastStatus < 300 {
			if strings.TrimSpace(lastCT) == "" {
				lastCT = "audio/wav"
			}
			return lastBody, lastCT, nil
		}

		// Retry on transient 5xx
		if lastStatus >= 500 && lastStatus < 600 {
			time.Sleep(time.Duration(1+attempt) * 2 * time.Second)
			continue
		}

		break
	}

	// If Yoruba primary model failed and it's not the facebook baseline, try facebook/mms-tts-yor as a fallback
	if (origLang == "yo" || origLang == "yo-ng" || origLang == "yor" || origLang == "yoruba") && model != "facebook/mms-tts-yor" {
		fbModel := "facebook/mms-tts-yor"
		apiURL = fmt.Sprintf("https://api-inference.huggingface.co/models/%s", fbModel)
		log.Printf("TTS fallback: lang=%s fallback_model=%s", origLang, fbModel)
		for attempt := 0; attempt < 2; attempt++ {
			req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
			if err != nil {
				break
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "audio/wav")
			req.Header.Set("User-Agent", "language-translator-backend/tts (+github.com/developia-II)")
			resp, err := client.Do(req)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			func() {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				lastStatus = resp.StatusCode
				lastBody = b
				lastCT = resp.Header.Get("Content-Type")
			}()
			log.Printf("TTS fallback HF response: status=%d ct=%s len=%d (attempt %d)", lastStatus, lastCT, len(lastBody), attempt+1)
			if lastStatus >= 200 && lastStatus < 300 {
				if strings.TrimSpace(lastCT) == "" {
					lastCT = "audio/wav"
				}
				return lastBody, lastCT, nil
			}
		}
	}

	preview := string(lastBody)
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	return nil, "", fmt.Errorf("huggingface %d: %s", lastStatus, preview)
}

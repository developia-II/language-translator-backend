package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"net/url"
	"strings"
	"time"
)

type LibreTranslateRequest struct {
    Q      string `json:"q"`
    Source string `json:"source"`
    Target string `json:"target"`
    Format string `json:"format,omitempty"`
    APIKey string `json:"api_key,omitempty"`
}

type LibreTranslateResponse struct {
    TranslatedText string `json:"translatedText"`
}

func TranslateText(text, sourceLang, targetLang string) (string, error) {
    // 1) Try MyMemory first (free, no API key; better support for Nigerian languages)
    if translated, err := translateWithMyMemory(text, sourceLang, targetLang); err == nil && translated != "" {
        return translated, nil
    }

    // 2) Fallback to LibreTranslate (public instance or configured URL)
    if translated, err := translateWithLibreTranslate(text, sourceLang, targetLang); err == nil && translated != "" {
        return translated, nil
    } else if err != nil {
        return "", fmt.Errorf("translation failed: %w", err)
    }

    return "", fmt.Errorf("all translation services failed")
}

// translateWithLibreTranslate posts to a LibreTranslate endpoint and returns translated text.
func translateWithLibreTranslate(text, sourceLang, targetLang string) (string, error) {
    // Normalize language tags
    s := strings.ToLower(strings.TrimSpace(sourceLang))
    t := strings.ToLower(strings.TrimSpace(targetLang))

    // Build endpoint list
    endpoints := []string{}
    if v := strings.TrimSpace(os.Getenv("LIBRETRANSLATE_URL")); v != "" {
        endpoints = append(endpoints, v)
    }
    // Known public mirrors (may be rate-limited; try multiple)
    endpoints = append(endpoints,
        "https://libretranslate.com/translate",
        "https://translate.argosopentech.com/translate",
        "https://libretranslate.de/translate",
    )

    apiKey := strings.TrimSpace(os.Getenv("LIBRETRANSLATE_API_KEY"))

    client := &http.Client{Timeout: 20 * time.Second}
    var lastErr error
    for _, apiURL := range endpoints {
        reqBody := LibreTranslateRequest{
            Q:      text,
            Source: s,
            Target: t,
            Format: "text",
        }
        if apiKey != "" {
            reqBody.APIKey = apiKey
        }

        jsonData, err := json.Marshal(reqBody)
        if err != nil {
            lastErr = err
            continue
        }

        httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
        if err != nil {
            lastErr = err
            continue
        }
        httpReq.Header.Set("Content-Type", "application/json")
        httpReq.Header.Set("Accept", "application/json")
        httpReq.Header.Set("User-Agent", "language-translator-backend/translator (+github.com/developia-II)")

        resp, err := client.Do(httpReq)
        if err != nil {
            lastErr = err
            continue
        }
        bodyBytes, _ := io.ReadAll(resp.Body)
        resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            // Try next mirror on non-200
            preview := string(bodyBytes)
            if len(preview) > 500 { preview = preview[:500] + "..." }
            lastErr = fmt.Errorf("libretranslate %d from %s: %s", resp.StatusCode, apiURL, preview)
            continue
        }

        var result LibreTranslateResponse
        if err := json.Unmarshal(bodyBytes, &result); err != nil {
            // Some mirrors return HTML (e.g., Cloudflare). Try next.
            preview := string(bodyBytes)
            if len(preview) > 500 { preview = preview[:500] + "..." }
            lastErr = fmt.Errorf("invalid JSON from %s: %v; body: %s", apiURL, err, preview)
            continue
        }

        if strings.TrimSpace(result.TranslatedText) != "" {
            return result.TranslatedText, nil
        }
        lastErr = fmt.Errorf("empty translation from %s", apiURL)
    }

    if lastErr != nil {
        return "", fmt.Errorf("translation failed: %w", lastErr)
    }
    return "", fmt.Errorf("translation failed: unknown error")
}

// translateWithMyMemory uses the public MyMemory API.
func translateWithMyMemory(text, sourceLang, targetLang string) (string, error) {
    // Build URL: https://api.mymemory.translated.net/get?q=...&langpair=src|tgt
    base := "https://api.mymemory.translated.net/get"
    q := url.Values{}
    q.Set("q", text)
    q.Set("langpair", fmt.Sprintf("%s|%s", sourceLang, targetLang))
    fullURL := fmt.Sprintf("%s?%s", base, q.Encode())

    resp, err := http.Get(fullURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    bodyBytes, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        preview := string(bodyBytes)
        if len(preview) > 500 {
            preview = preview[:500] + "..."
        }
        return "", fmt.Errorf("mymemory %d: %s", resp.StatusCode, preview)
    }

    var mm struct {
        ResponseData struct {
            TranslatedText string `json:"translatedText"`
        } `json:"responseData"`
        ResponseStatus  int    `json:"responseStatus"`
        ResponseDetails string `json:"responseDetails"`
    }

    if err := json.Unmarshal(bodyBytes, &mm); err != nil {
        preview := string(bodyBytes)
        if len(preview) > 500 {
            preview = preview[:500] + "..."
        }
        return "", fmt.Errorf("invalid JSON from mymemory: %v; body: %s", err, preview)
    }

    if mm.ResponseStatus == 200 && mm.ResponseData.TranslatedText != "" {
        return mm.ResponseData.TranslatedText, nil
    }

    if mm.ResponseDetails != "" {
        return "", fmt.Errorf("mymemory error: %s", mm.ResponseDetails)
    }

    return "", fmt.Errorf("mymemory returned empty translation")
}

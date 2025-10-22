package services

import (
	"bytes"
	"fmt"
	"os/exec"
)

// ESpeakTTS uses eSpeak-NG for text-to-speech
func ESpeakTTS(text, lang string) ([]byte, string, error) {
	// Map language codes to eSpeak voices
	voice := map[string]string{
		"yo-NG": "yoruba",
		"ig-NG": "igbo",
		"ha-NG": "hausa",
	}[lang]
	if voice == "" {
		voice = "en" // fallback to English
	}

	cmd := exec.Command("espeak-ng",
		"-s", "160", // Speed (words per minute)
		"-p", "50", // Pitch adjustment (0-99)
		"-a", "100", // Amplitude (volume)
		"-v", voice,
		"--stdout",
		text,
	)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, "", fmt.Errorf("espeak-ng failed: %s - %w", stderr.String(), err)
	}

	return out.Bytes(), "audio/wav", nil
}

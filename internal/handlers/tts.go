package handlers

import (
	"log"
	"os"
	"strings"

	"github.com/developia-II/language-translator-backend/internal/services"
	"github.com/developia-II/language-translator-backend/utils"
	"github.com/gofiber/fiber/v2"
)

type ttsReq struct {
	Text string `json:"text"`
	Lang string `json:"lang"`
}

func normalizeLang(l string) string {
	l = strings.TrimSpace(l)
	l = strings.ReplaceAll(l, "_", "-")
	l = strings.ToLower(l)
	switch l {
	case "yo", "yo-ng":
		return "yo-NG"
	case "ig", "ig-ng":
		return "ig-NG"
	case "ha", "ha-ng":
		return "ha-NG"
	case "en", "en-ng":
		return "en-NG"
	default:

		if strings.Contains(l, "-") {
			parts := strings.SplitN(l, "-", 2)
			return strings.ToLower(parts[0]) + "-" + strings.ToUpper(parts[1])
		}
		return strings.ToLower(l)
	}
}

func TTS(c *fiber.Ctx) error {
	var req ttsReq
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if strings.TrimSpace(req.Text) == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "text is required")
	}

	lang := normalizeLang(req.Lang)

	var audio []byte
	var ctype string
	var err error

	// Priority: eSpeak > Google > ElevenLabs (YO/IG/HA) > Hugging Face MMS
	if os.Getenv("USE_ESPEAK") == "true" {
		log.Printf("TTS handler: provider=eSpeak lang=%s", lang)
		audio, ctype, err = services.ESpeakTTS(req.Text, lang)
	} else if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		// audio, ctype, err = services.GoogleCloudTTS(req.Text, lang)
	} else if ak := os.Getenv("ELEVENLABS_API_KEY"); strings.TrimSpace(ak) != "" {
		// Only route certain languages to ElevenLabs if API key is present
		low := strings.ToLower(lang)
		base := low
		if i := strings.IndexByte(low, '-'); i > 0 {
			base = low[:i]
		}
		switch base {
		case "yo", "ig", "ha":
			log.Printf("TTS handler: provider=ElevenLabs lang=%s", lang)
			audio, ctype, err = services.ElevenLabsTTS(req.Text, lang)
		default:
			// fall through to HF for other languages even when ElevenLabs key is set
			log.Printf("TTS handler: provider=HuggingFace (fallback from ElevenLabs) lang=%s", lang)
			audio, ctype, err = services.SynthesizeTTS(req.Text, lang)
		}
	} else {
		log.Printf("TTS handler: provider=HuggingFace lang=%s", lang)
		audio, ctype, err = services.SynthesizeTTS(req.Text, lang)
	}

	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadGateway, "TTS failed: "+err.Error())
	}

	c.Set("Content-Type", ctype)
	c.Set("Cache-Control", "no-store")
	return c.Send(audio)
}

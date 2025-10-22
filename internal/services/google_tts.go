package services

// import (
// 	"context"
// 	"fmt"
// 	"os"

// 	texttospeech "cloud.google.com/go/texttospeech/apiv1"
// 	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
// )

// // GoogleCloudTTS calls Google Cloud Text-to-Speech API
// func GoogleCloudTTS(text, lang string) ([]byte, string, error) {
// 	ctx := context.Background()

// 	// Initialize client with credentials from environment
// 	client, err := texttospeech.NewClient(ctx)
// 	if err != nil {
// 		return nil, "", fmt.Errorf("failed to create client: %w", err)
// 	}
// 	defer client.Close()

// 	// Configure voice for Nigerian languages
// 	voice := &texttospeechpb.VoiceSelectionParams{
// 		LanguageCode: lang,
// 		SsmlGender:   texttospeechpb.SsmlVoiceGender_FEMALE,
// 	}

// 	// Configure audio output
// 	audioConfig := &texttospeechpb.AudioConfig{
// 		AudioEncoding: texttospeechpb.AudioEncoding_MP3,
// 		SpeakingRate: 1.0,
// 	}

// 	// Perform text-to-speech
// 	resp, err := client.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
// 		Input: &texttospeechpb.SynthesisInput{
// 			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
// 		},
// 		Voice:       voice,
// 		AudioConfig: audioConfig,
// 	})
// 	if err != nil {
// 		return nil, "", fmt.Errorf("SynthesizeSpeech: %w", err)
// 	}

// 	return resp.AudioContent, "audio/mpeg", nil
// }

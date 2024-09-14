package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/line/line-bot-sdk-go/linebot"
	"google.golang.org/api/option"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.Background()

	bot, err := linebot.New(os.Getenv("LINE_CHANNEL_SECRET"), os.Getenv("LINE_CHANNEL_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	callbackHandler := func(w http.ResponseWriter, r *http.Request) {
		events, err := bot.ParseRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					response, err := callGemini(ctx, message.Text)
					if err != nil {
						log.Println(err)
						bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("出現問題，請稍後再試")).Do()
						return
					}
					bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(response)).Do()
				}
			}
		}
	}

	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server is running on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func callGemini(ctx context.Context, message string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Get the model
	model := client.GenerativeModel("gemini-1.5-flash")

	// Create a prompt
	prompt := message

	// Generate content
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatal(err)
	}

	// Print the response
	log.Printf("Response: %#v\n", resp)

	content, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return "", errors.New("failed to get text from response")
	}

	return string(content), nil
}

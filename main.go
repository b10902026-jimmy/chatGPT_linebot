package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {
	// Initialize log file
	f, err := os.OpenFile("app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	// Load .env file
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize Line Bot SDK
	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal("Error initializing Line Bot:", err)
	}

	// Set up HTTP server
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		log.Println("Received a callback request.")
		events, err := bot.ParseRequest(req)
		if err != nil {
			log.Println("Error parsing request:", err)
			http.Error(w, "Can't parse request", http.StatusBadRequest)
			return
		}

		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					log.Printf("Received a message: %s\n", message.Text)
					gptResponse, err := fetchGPTResponse(message.Text)
					if err != nil {
						log.Println("Error fetching GPT-3 response:", err)
						return
					}
					log.Printf("GPT-3 response: %s\n", gptResponse)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(gptResponse)).Do(); err != nil {
						log.Println("Error sending reply:", err)
					}
				}
			}
		}
	})

	log.Println("Server is running on port 30019")
	log.Fatal(http.ListenAndServe(":30019", nil))
}

func fetchGPTResponse(prompt string) (string, error) {
	client := resty.New()
	gptAPIKey := os.Getenv("GPT_API_KEY")

	payload := map[string]interface{}{
		"prompt":     prompt,
		"max_tokens": 100,
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+gptAPIKey).
		SetBody(payload).
		Post("https://api.openai.com/v1/engines/davinci-codex/completions")

	if err != nil {
		log.Println("Error making GPT-3 API call:", err)
		return "", err
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body(), &result)
	choices := result["choices"].([]interface{})
	firstChoice := choices[0].(map[string]interface{})
	message := firstChoice["message"].(map[string]interface{})
	content := message["content"].(string)

	return content, nil
}

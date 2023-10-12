package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/line/line-bot-sdk-go/linebot"
)

func LineBotHandler(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/callback":
		handleCallback(w, req)
	case "/":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, This is a chatGPT line bot!"))
	case "/favicon.ico":
		w.WriteHeader(http.StatusOK)
		http.ServeFile(w, req, "open-ai.ico")
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func handleCallback(w http.ResponseWriter, req *http.Request) {
	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Println("Error initializing Line Bot:", err)
		return
	}

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
				var gptResponse string
				var err error

				if message.Text == "你是誰" {
					log.Println("Processing '你是誰'")
					gptResponse = "我是由施鈞譯jimmy架設的自動回覆機器人，使用gpt3.5-turbo作為語言模型"
					log.Println("Processed '你是誰', ready to reply.")
				} else {
					log.Println("Processing a GPT-3 request.")
					gptResponse, err = fetchGPTResponse("使用繁體中文回答：" + message.Text)
					if err != nil {
						log.Println("Error fetching GPT-3 response:", err)
						return
					}
					log.Println("Processed GPT-3 request, ready to reply.")
				}

				log.Printf("GPT-3 response: %s\n", gptResponse)
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(gptResponse)).Do(); err != nil {
					log.Println("Error sending reply:", err)
				}
			}
		}
	}
}

func fetchGPTResponse(prompt string) (string, error) {
	client := resty.New()
	client.SetTimeout(10 * time.Second) // Added a 10-second timeout
	gptAPIKey := os.Getenv("GPT_API_KEY")

	payload := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": prompt},
		},
	}

	log.Println("Sending GPT-3 API call")
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+gptAPIKey).
		SetBody(payload).
		Post("https://api.openai.com/v1/chat/completions")
	log.Println("Received GPT-3 API response")

	if err != nil {
		log.Println("Error making GPT-3 API call:", err)
		return "", err
	}

	if resp.IsError() {
		log.Printf("API returned error: %v\n", resp)
		return "", fmt.Errorf("API returned status code %v", resp.StatusCode())
	}

	var result map[string]interface{}
	if err = json.Unmarshal(resp.Body(), &result); err != nil {
		log.Println("Error unmarshaling response:", err)
		return "", err
	}

	if choices, ok := result["choices"].([]interface{}); ok {
		if len(choices) > 0 {
			if firstChoice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := firstChoice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						return content, nil
					}
				}
			}
		}
	}

	log.Println("Unexpected response format.")
	return "", fmt.Errorf("unexpected response format")
}

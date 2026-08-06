// Package telegram sends formatted alert messages to a Telegram chat via the
// Bot API and records the outcome through the notification repository.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const apiBaseURL = "https://api.telegram.org"

// Client is a minimal Telegram Bot API client for sending chat messages and polling updates.
type Client struct {
	httpClient    *http.Client
	pollingClient *http.Client
}

// NewClient builds a Telegram API client with bounded request timeouts.
func NewClient() *Client {
	return &Client{
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		pollingClient: &http.Client{Timeout: 35 * time.Second},
	}
}

type sendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type sendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type getUpdatesResponse struct {
	OK          bool     `json:"ok"`
	Result      []Update `json:"result"`
	Description string   `json:"description,omitempty"`
}

// SendMessage sends text as an HTML-formatted message to chatID using botToken.
func (c *Client) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	if botToken == "" || chatID == "" {
		return fmt.Errorf("telegram bot token or chat id is not configured")
	}

	payload, err := json.Marshal(sendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
	if err != nil {
		return fmt.Errorf("marshalling telegram request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBaseURL, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading telegram response: %w", err)
	}

	var result sendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("unmarshalling telegram response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}
	return nil
}

// GetUpdates polls Telegram for updates starting from offset.
func (c *Client) GetUpdates(ctx context.Context, botToken string, offset, timeout int) ([]Update, error) {
	if botToken == "" {
		return nil, fmt.Errorf("telegram bot token is not configured")
	}

	url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=%d", apiBaseURL, botToken, offset, timeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building getUpdates request: %w", err)
	}

	resp, err := c.pollingClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polling telegram updates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading getUpdates response: %w", err)
	}

	var result getUpdatesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling getUpdates response: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram API error: %s", result.Description)
	}
	return result.Result, nil
}


package telegram_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/t4ke0/telegram_bridge/pkg/db"
)

var botURL string = os.Getenv("BOT_URL")

// Updates
type Updates struct {
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  struct {
			ID   int `json:"message_id"`
			From struct {
				UserID int `json:"id"`
			} `json:"from"`
			Text string `json:"text"`
			Chat struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"result"`
}

// Command
type Command struct {
	updateID int
	chatID   int

	userID   int
	username string

	root string
	args []string
}

// TelegramClient
type TelegramClient struct {
	baseURL string

	httpClient *http.Client
}

// NewTelegramClient
func NewTelegramClient() *TelegramClient {
	return &TelegramClient{
		baseURL:    botURL,
		httpClient: new(http.Client),
	}
}

func (t *TelegramClient) makeRequest(method, url string, data []byte) (*http.Response, error) {
	body := &bytes.Buffer{}
	if data != nil {
		body = bytes.NewBuffer(data)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	return t.httpClient.Do(req)
}

// UpdateWorker
func (t *TelegramClient) UpdateWorker(errC chan error) <-chan Command {

	out := make(chan Command)
	go func() {
		defer close(out)
		url := fmt.Sprintf("%s/getUpdates", t.baseURL)
		var updateID int
		for {
			resp, err := t.makeRequest(http.MethodGet, url, nil)
			if err != nil {
				errC <- err
				continue
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				errC <- err
				continue
			}
			defer resp.Body.Close()

			var updates Updates
			if err := json.Unmarshal(data, &updates); err != nil {
				errC <- err
				continue
			}

			lastIndex := len(updates.Result) - 1
			result := updates.Result[lastIndex]
			uID := result.UpdateID
			if uID == updateID {
				continue
			}

			updateID = uID

			var args []string

			splitted := strings.Fields(result.Message.Text)
			if len(splitted) == 0 {
				continue
			}

			if len(splitted) > 1 {
				args = splitted[1:]
			}

			out <- Command{
				updateID: result.UpdateID,
				chatID:   result.Message.Chat.ID,
				userID:   result.Message.From.UserID,
				username: result.Message.Chat.Username,
				root:     splitted[0],
				args:     args,
			}
		}
	}()

	return out
}

// HandleUpdate
func (t *TelegramClient) HandleUpdate(cmd <-chan Command, errC chan error) (err error) {

	conn, err := db.New()
	if err != nil {
		err = err
		return
	}

	go func() {
		defer func() {
			if err := conn.Close(); err != nil {
				errC <- err
			}
		}()
		for c := range cmd {
			lastID, err := conn.GetLastUpdateID()
			if err != nil {
				errC <- err
				continue
			}

			if lastID == c.updateID {
				continue
			}

			if err := conn.InsertUpdateID(c.updateID); err != nil {
				errC <- err
			}

			switch c.root {
			case "/getid":
				if err := t.SendMessageHook(c.chatID, strconv.Itoa(c.userID)); err != nil {
					errC <- err
				}
			case "/subscribe":

				token, err := conn.InsertNewUser(strconv.Itoa(c.userID), c.username)
				if err == db.ErrConflict {
					if err := t.SendMessageHook(c.chatID, "already subbed"); err != nil {
						errC <- err
					}
					continue
				}

				if err := t.SendMessageHook(c.chatID, fmt.Sprintf("your token to use with the bridge %v", token)); err != nil {
					errC <- err
				}
			}

		}
	}()

	return
}

// SendMessageHook
func (t *TelegramClient) SendMessageHook(chatID int, text string) error {
	url := fmt.Sprintf("%s/sendMessage?chat_id=%d&text=%s", t.baseURL, chatID, text)
	resp, err := t.makeRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[%v] failed to send message via telegram", resp.StatusCode)
	}

	return nil
}

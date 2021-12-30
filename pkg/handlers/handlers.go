package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/t4ke0/telegram_bridge/pkg/db"
	"github.com/t4ke0/telegram_bridge/pkg/telegram_api"
)

// SubscribeRequest
type SubscribeRequest struct {
	TelegramUserID string `json:"telegram_user_id"`
	Username       string `json:"username"`
}

func errorHandler(w http.ResponseWriter) {
	if r := recover(); r != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error %v", r)
		return
	}
}

// HandleSubscribe
func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer errorHandler(w)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	defer func() {
		defer errorHandler(w)
		if err := r.Body.Close(); err != nil {
			panic(err)
		}
	}()

	var req SubscribeRequest
	if err := json.Unmarshal(data, &req); err != nil {
		panic(err)
	}

	if strings.TrimSpace(req.TelegramUserID) == "" || strings.TrimSpace(req.Username) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := db.New()
	if err != nil {
		panic(err)
	}

	defer func() {
		defer errorHandler(w)
		if err := conn.Close(); err != nil {
			panic(err)
		}
	}()

	token, err := conn.InsertNewUser(req.TelegramUserID, req.Username)
	if err == db.ErrConflict {
		w.WriteHeader(http.StatusConflict)
		return
	}

	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")

	resp := struct {
		Token string `json:"token"`
	}{
		Token: token,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}

// SendMessageRequest
type SendMessageRequest struct {
	TextMessage string `json:"text_message"`
}

const tokenHeaderKey string = "X-API-Key"

// HandleSendMessage
func HandleSendMessage(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer errorHandler(w)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	var req SendMessageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		panic(err)
	}

	token := r.Header.Get(tokenHeaderKey)

	//
	conn, err := db.New()
	if err != nil {
		panic(err)
	}

	defer func() {
		defer errorHandler(w)
		if err := conn.Close(); err != nil {
			panic(err)
		}
	}()
	//

	s, err := conn.GetSubscriber(token)
	if err != nil {
		panic(err)
	}

	if err := conn.InsertNewMessage(strconv.Itoa(s.ID), req.TextMessage); err != nil {
		panic(err)
	}

	telegramClient := telegram_api.NewTelegramClient()

	if err := telegramClient.SendMessageHook(s.ID, req.TextMessage); err != nil {
		panic(err)
	}
}

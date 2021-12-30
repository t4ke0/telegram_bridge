package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"

	"github.com/t4ke0/telegram_bridge/pkg/db"
	"github.com/t4ke0/telegram_bridge/pkg/handlers"
	"github.com/t4ke0/telegram_bridge/pkg/telegram_api"
)

var (
	portNumber string = os.Getenv("PORT")
)

func init() {
	if err := db.InitDB(); err != nil {
		log.Fatal(err)
	}
	if portNumber == "" {
		portNumber = "8080"
	}
}

func main() {

	errC := make(chan error)

	go func() {
		for {
			select {
			case err := <-errC:
				log.Printf("[Error] %v", err)
			default:
			}
		}
	}()

	go func() {
		log.Printf("running telegram bridge listener ...")
		client := telegram_api.NewTelegramClient()
		updates := client.UpdateWorker(errC)
		if err := client.HandleUpdate(updates, errC); err != nil {
			errC <- err
		}
	}()

	//
	http.HandleFunc("/api/subscribe",
		handlers.Middlewares{handlers.LogginMiddleware}.
			Chain(handlers.HandleSubscribe))

	http.HandleFunc("/api/send/message",
		handlers.Middlewares{handlers.LogginMiddleware, handlers.AuthorizeTokenMiddleware}.
			Chain(handlers.HandleSendMessage))
	//

	log.Printf("listening on 127.0.0.1:%s", portNumber)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", portNumber), nil); err != nil {
		log.Fatal(err)
	}

}

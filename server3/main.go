package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("failed to accept websocket: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")

	ctx := r.Context()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		fmt.Printf("Received: %s\n", string(data))
		_ = conn.Write(ctx, websocket.MessageText, data) // Echo back
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fmt.Println("WebSocket Server berjalan di ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

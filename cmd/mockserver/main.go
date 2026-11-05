package main

import (
	// "context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Simple mock gRPC server implementation using net/rpc or standard grpc stub if needed,
// or we can run an HTTP/2 gRPC server with reflection.
type MockEchoServer struct {
	// UnimplementedMockServer embedded if generated
}

func main() {
	// 1. Start WebSocket Mock Server on :8080
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
				InsecureSkipVerify: true,
			})
			if err != nil {
				log.Printf("failed to accept websocket: %v", err)
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "bye")

			ctx := r.Context()
			log.Println("WebSocket client connected")

			_ = conn.Write(ctx, websocket.MessageText, []byte(`{"event":"welcome","message":"Hello from TermAPI Mock WS Server"}`))

			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			go func() {
				count := 0
				for {
					select {
					case <-ctx.Done():
						return
					case t := <-ticker.C:
						count++
						msg := fmt.Sprintf(`{"tick":%d,"time":"%s"}`, count, t.Format(time.RFC3339))
						_ = conn.Write(ctx, websocket.MessageText, []byte(msg))
					}
				}
			}()

			for {
				typ, data, err := conn.Read(ctx)
				if err != nil {
					break
				}
				log.Printf("WS Received: %s", string(data))
				_ = conn.Write(ctx, typ, data)
			}
		})

		log.Println("Mock WebSocket server listening on :8080 (/ws)...")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("WS server error: %v", err)
		}
	}()

	// 2. Start gRPC Mock Server on :50051 with Reflection
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen on :50051: %v", err)
		}

		s := grpc.NewServer()
		reflection.Register(s)

		log.Println("Mock gRPC server listening on :50051 (Reflection enabled)...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	select {}
}

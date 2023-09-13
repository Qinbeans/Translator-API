package protogo

import (
	"context"
	"encoding/json"
	"log"
	"time"
	"translator-api/utils"

	// fasthttp
	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	MAX_QUEUE_SIZE = 100
)

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type NaiveRequest struct {
	Text string `json:"text"`
}

type Queue struct {
	Requests chan *TranslateRequest
}

func (q *Queue) Push(req *TranslateRequest) {
	q.Requests <- req
}

func (q *Queue) Pop() *TranslateRequest {
	return <-q.Requests
}

func NewQueue() *Queue {
	return &Queue{Requests: make(chan *TranslateRequest, MAX_QUEUE_SIZE)}
}

type ProtoGo struct {
	queue      *Queue
	tranClient TranslatorClient
	conn       *grpc.ClientConn
	wsConns    map[string]*websocket.Conn
}

func NewProtoGo(addr string) *ProtoGo {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	return &ProtoGo{
		queue:      NewQueue(),
		conn:       conn,
		tranClient: NewTranslatorClient(conn),
		wsConns:    make(map[string]*websocket.Conn),
	}
}

func (p *ProtoGo) Start() {
	go func() {
		for {
			req := p.queue.Pop()
			_, err := p.tranClient.Translate(context.Background(), req)
			if err != nil {
				log.Printf("Error: %s", err)
			}
			if ws, ok := p.wsConns[req.Details.Token]; ok {
				if ws != nil {
					if err != nil {
						ws.WriteJSON(map[string]string{"status": "error", "result": err.Error()})
					} else {
						ws.WriteJSON(map[string]string{"status": "done", "result": req.Details.Message})
					}
				}
			}
		}
	}()
}

func (p *ProtoGo) Close() {
	p.conn.Close()
}

func (p *ProtoGo) HandleWebSocket(ctx *fasthttp.RequestCtx) {
	token := ctx.UserValue("token").(string)
	if token == "" {
		utils.SendJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if ws, ok := p.wsConns[token]; ok {
		if ws != nil {
			ws.Close()
			println("Closing old connection")
		}

		err := upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
			p.wsConns[token] = ws
			defer func() {
				delete(p.wsConns, token)
				ws.Close()
			}()

			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			timeout := time.NewTimer(5 * time.Minute)

			ws.WriteJSON(map[string]string{"status": "connected"})
			for {
				select {
				case <-ticker.C:
					if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
						log.Println("Ping error:", err)
						return
					}
				case <-timeout.C:
					log.Println("Timeout")
					return
				}
			}
		})
		if err != nil {
			log.Println("upgrade:", err)
			return
		}
	}

	log.Println("Invalid token")
}

func (p *ProtoGo) Translate(ctx *fasthttp.RequestCtx) {

	//check if queue is full
	if len(p.queue.Requests) >= MAX_QUEUE_SIZE {
		utils.SendJSON(ctx, fasthttp.StatusTooManyRequests, map[string]string{"error": "queue is full"})
		return
	}

	var req NaiveRequest
	err := json.Unmarshal(ctx.PostBody(), &req)
	if err != nil || req.Text == "" {
		utils.SendJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	target := uuid.New().String()

	p.queue.Push(&TranslateRequest{
		Text: req.Text,
		Details: &Details{
			Token:   target,
			Message: "",
		},
	})

	p.wsConns[target] = nil

	utils.SendJSON(ctx, fasthttp.StatusOK, map[string]string{"target": target})
}
package protogo

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"
	"translator-api/utils"

	// fasthttp
	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
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

// Start the queue worker
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

// Handle websocket connection
//
// GET /ws/{token}
//
// response:
//
//	{
//		"status": "connected"
//	}
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

// Translate text
//
// POST /translate
//
// body:
//
//	{
//		"text": "Hello world"
//	}
//
// response:
//
//	{
//		"token": "uuid"
//	}
func (p *ProtoGo) Translate(ctx *fasthttp.RequestCtx) {
	log.Println("POST:\t/translate")

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

	utils.SendJSON(ctx, fasthttp.StatusOK, map[string]string{"token": target})
}

// Health check
//
// GET /health
//
// response:
//
//	{
//		"status": "ok",
//		"queue_size": 0,
//		"ws_connections": 0,
//		"grpc": "ready"
//	}
func (p *ProtoGo) Health(ctx *fasthttp.RequestCtx) {
	log.Println("GET:\t/health")

	//send json response
	health_status := map[string]string{}

	health_status["status"] = "ok"
	health_status["queue_size"] = strconv.Itoa(len(p.queue.Requests))
	health_status["ws_connections"] = strconv.Itoa(len(p.wsConns))
	// check connection to grpc server
	if p.conn.GetState() == connectivity.Connecting {
		health_status["grpc"] = "connecting"
	} else if p.conn.GetState() == connectivity.Ready {
		health_status["grpc"] = "ready"
	} else if p.conn.GetState() == connectivity.Idle {
		health_status["grpc"] = "idle"
	} else {
		health_status["grpc"] = "error"
	}

	utils.SendJSON(ctx, fasthttp.StatusOK, health_status)
}

package main

import (
	"fmt"
	"log"
	"os"
	protogo "translator-api/proto-go"

	"github.com/georgecookeIW/fasthttprouter"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	grpc_host := os.Getenv("GRPC_HOST")
	grpc_port := os.Getenv("GRPC_PORT")

	connection_str := fmt.Sprintf("%s:%s", host, port)
	grpc_connection_str := fmt.Sprintf("%s:%s", grpc_host, grpc_port)

	router := fasthttprouter.New()

	fmt.Println("Connecting to gRPC server at " + grpc_connection_str)

	requester := protogo.NewProtoGo(grpc_connection_str)
	defer requester.Close()
	requester.Start()

	fmt.Println("Connect to http://" + connection_str)

	router.HandleOPTIONS = true

	router.HandleCORS = fasthttprouter.CORS{
		Handle:       true,
		AllowOrigin:  "https://canto.qinbeans.net",
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "X-Requested-With"},
		MaxAge:       3600,
	}

	router.GET("/api/v1/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("Nothing to see here.")
	})
	router.GET("/api/v1/health", requester.Health)
	router.GET("/api/v1/ws/:token", requester.HandleWebSocket)
	router.POST("/api/v1/translate", requester.Translate)

	log.Fatal(fasthttp.ListenAndServe(connection_str, router.Handler))
}

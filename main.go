package main

import (
	"fmt"
	"log"
	"os"
	protogo "translator-api/proto-go"
	"translator-api/utils"

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

	router.GET("/v1/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("Nothing to see here.")
	})
	router.GET("/v1/health", func(ctx *fasthttp.RequestCtx) {
		//send json response
		utils.SendJSON(ctx, 200, map[string]string{"status": "ok"})
	})
	router.GET("/v1/ws/:token", requester.HandleWebSocket)
	router.POST("/v1/translate", requester.Translate)

	log.Fatal(fasthttp.ListenAndServe(connection_str, router.Handler))
}

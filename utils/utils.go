package utils

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

func SendJSON(ctx *fasthttp.RequestCtx, status int, body any) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		ctx.SetStatusCode(500)
		ctx.SetBodyString("{\"error\": \"Internal Server Error\"}")
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	ctx.SetBody(bodyJSON)
}

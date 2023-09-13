# Translator API

## Description

Uses [TransCan](https://github.com/ayaka14732/TransCan.git) to translate text from English to Contonese. The API is hosted using Golang and the FastHTTP framework. Golang communicates with the Python server via ProtoBuf. The Python server uses the PyTorch, JAX, Transformers, etc. libraries to translate the text.

## Usage

### Manualy starting

- Download the weights from TransCan and place them in the `proto-py/data` folder
- Create a `.env` in the project root
  - Add `PORT=80` to the `.env` file
  - Add `HOST=localhost` to the `.env` file
  - Add `GRPC_PORT=50051` to the `.env` file
  - Add `GRPC_HOST=localhost` to the `.env` file
- Start the Python server by running `python3 proto-py/server.py`
- Start the Golang server by running `go run main.go`

### Using the runner script

- Run `./start start` to start the servers
- Run `./start stop` to stop the servers

### Testing

- Can use curl to test the API, e.g. `curl -X POST -H "Content-Type: application/json" -d '{"text":"hello"}' http://localhost:8081/v1/translate`
  - The response should be `{"status": "succss","text":"你好"}` or something similar once connected to the websocket

## Notes

Meant for use with my Translator frontend

## TODO

- [ ] Add tests
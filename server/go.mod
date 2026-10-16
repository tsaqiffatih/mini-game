module github.com/tsaqiffatih/mini-game

go 1.26.0

replace github.com/tsaqiffatih/mini-game => ../

require (
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/notnil/chess v1.10.0
	golang.org/x/time v0.16.0
)

require github.com/felixge/httpsnoop v1.0.3 // indirect

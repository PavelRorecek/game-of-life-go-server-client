package main

import (
	"pavelrorecek.com/ebitenginetest/client"
	"pavelrorecek.com/ebitenginetest/server"
)

func main() {
	go server.StartServer()
	client.StartClient()
}

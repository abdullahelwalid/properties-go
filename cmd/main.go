package main

import (
	"golang-test/api"
	"golang-test/config"
)

func main() {
	config.LoadConfig()
	api.StartServer()
}

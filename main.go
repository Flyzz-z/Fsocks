package main

import (
	"Fsocks/config"
	"Fsocks/listener"
	"log"
)

func main() {
	err := config.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}
	err = listener.Listen(7890)
	if err != nil {
		log.Fatal(err)
	}
}

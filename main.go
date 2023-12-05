package main

import (
	"chatcard/admin"
	"chatcard/proxy"
	"fmt"
	"log"
	"sync"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("chatcard...")
	wg := sync.WaitGroup{}
	wg.Add(1)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	go proxy.Run(":5200", nil)
	go admin.Run(":5201", nil)
	wg.Wait()
}

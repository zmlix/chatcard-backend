package main

import (
	"chatcard/proxy"
	"fmt"
)

func main() {
	fmt.Println("chatcard...")
	proxy.Run(":5200", nil)
}

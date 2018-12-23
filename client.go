package main

import (
	"fmt"
	"net"
)

func main() {
	fmt.Println(net.Dial("tcp", net.JoinHostPort("localhost", "8000")))
	fmt.Println("Idk bruh connect to the server and figure out how to exchange information somehow")
}

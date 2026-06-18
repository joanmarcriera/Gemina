package main

import (
	"fmt"

	"continuity-vpn/internal/bootstrap"
)

func main() {
	fmt.Println(bootstrap.ComponentStage("entitlement-api"))
}

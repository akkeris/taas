package utils

import (
	"fmt"
	"os"
)

func PrintDebug(msg interface{}) {
	if os.Getenv("DEBUG") != "" {
		fmt.Println(msg)
	}
}

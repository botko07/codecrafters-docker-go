package util

import (
	"fmt"
	"os"
)

func ExitIfErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

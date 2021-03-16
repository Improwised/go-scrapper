package util

import (
	"fmt"
	"os"
	"io"
)

func CheckError(err error) {
  if err != nil {
    if err == io.EOF {
      return
    }
    fmt.Println("Fatal error ", err.Error())
    os.Exit(1)
  }
}
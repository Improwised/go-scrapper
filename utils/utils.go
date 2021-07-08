package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func CheckError(err error, scrapStatus string) {
	if err != nil {
		if err == io.EOF {
			return
		}
		if err.Error() == "json: cannot unmarshal bool into Go struct field MatchServiceResponse.winner of type int" {
			scrapStatus = "NO_SEARCH_RESULTS"
			return
		}
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

func WriteDataToFileAsJSON(data interface{}, file *os.File) (int, error) {
	//write data as buffer to json encoder
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)

	err := encoder.Encode(data)
	if err != nil {
		return 0, err
	}
	n, err := file.Write(buffer.Bytes())
	if err != nil {
		return 0, err
	}
	return n, nil
}

package report

import (
	"encoding/json"
	"fmt"
	"os"
)

func WriteJSON(path string, rep *Report) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if path == "" || path == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Encode(rep *Report) ([]byte, error) {
	return json.MarshalIndent(rep, "", "  ")
}

func MustEncode(rep *Report) []byte {
	b, err := Encode(rep)
	if err != nil {
		panic(fmt.Sprintf("encode report: %v", err))
	}
	return b
}

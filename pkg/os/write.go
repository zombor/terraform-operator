package os

import (
	"fmt"
	"os"
	"strings"
)

func Write(id string, files []string) error {
	var err error = os.MkdirAll(fmt.Sprintf("/tmp/%s", id), os.ModePerm)
	if err != nil {
		return err
	}

	return os.WriteFile(fmt.Sprintf("/tmp/%s/main.tf", id), []byte(strings.Join(files, "\n")), 0644)
}

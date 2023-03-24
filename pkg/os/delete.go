package os

import (
	"fmt"
	"os"
)

func Delete(id string) error {
	return os.RemoveAll(fmt.Sprintf("/tmp/%s", id))
}

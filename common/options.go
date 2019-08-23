package options

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func ParseOptions(s string) (map[string]interface{}, error) {
	var result map[string]interface{}

	if len(s) == 0 {
		return nil, errors.New("Empty options")
	}

	decoder := json.NewDecoder(strings.NewReader(s))
	if err := decoder.Decode(&result); err != nil {
		fmt.Errorf("Error decoding JSON %q", err)
		return nil, err
	}

	return result, nil
}

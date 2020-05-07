package options

import (
	"encoding/json"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	"strings"
)

func ParseServerOptions(s string) (params map[string]map[string]interface{}, err error) {
	result := make(map[string]map[string]interface{})

	if len(s) == 0 {
		return result, nil
	}

	decoder := json.NewDecoder(strings.NewReader(s))
	if err := decoder.Decode(&result); err != nil {
		log.Errorf("Error decoding JSON %q", err)
		return nil, err
	}

	return result, nil
}

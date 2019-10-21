package options

import (
	"encoding/json"
	"errors"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
	interconv "github.com/mufti1/interconv/package"
	"strings"
)

func ParseOptions(s string) (map[string]interface{}, error) {
	var result map[string]interface{}

	if len(s) == 0 {
		return map[string]interface{}{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(s))
	if err := decoder.Decode(&result); err != nil {
		log.Errorf("Error decoding JSON %q", err)
		return nil, err
	}

	return result, nil
}

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

func CoerceToString(futureString interface{}) (string, error) {
		var result string

		switch futureString.(type) {
		case string:
			var icerr error
			result, icerr = interconv.ParseString(futureString)
			if icerr != nil {
				return "", icerr
			}
			return result, nil
		default:
			return "", errors.New("unable to coerce empty interface to string")
		}
}
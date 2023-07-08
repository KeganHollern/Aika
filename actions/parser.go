package actions

import (
	"encoding/json"
	"errors"
	"fmt"
)

func argsToMap(args string) (map[string]interface{}, error) {
	var msgMapTemplate interface{}
	err := json.Unmarshal([]byte(args), &msgMapTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal args; %w", err)
	}
	msgMap, ok := msgMapTemplate.(map[string]interface{})
	if !ok {
		return nil, errors.New("failed to cast args to map")
	}
	return msgMap, nil
}

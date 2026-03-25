package vibeutil

import "encoding/json"

// JSONMap is a convenience type for working with arbitrary JSON objects.
type JSONMap = map[string]interface{}

// ParseBlocksData parses a raw JSON message into a slice of JSONMap,
// representing the block-based content structure used by content nodes.
func ParseBlocksData(raw json.RawMessage) ([]JSONMap, error) {
	var blocks []JSONMap
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// ToJSON marshals any value into a json.RawMessage.
// Returns null JSON on marshal failure.
func ToJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return data
}

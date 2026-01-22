package shared

import "encoding/json"

// ToJSON converts a slice of CustomField to its JSON string representation.
func (cf CustomFields) ToJSON() (string, error) {
	if cf == nil {
		return "[]", nil
	}
	b, err := json.Marshal(cf)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

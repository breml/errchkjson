package example

import (
	"encoding/json"
)

func Issue47() {
	auth := struct {
		Auth string `json:"auth"`
	}{Auth: "123"}
	body, _ := json.Marshal(auth)
	_ = body
}

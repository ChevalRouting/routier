package requests

import (
	"encoding/json"
	"net/http"
)

func Decode[T any](r *http.Request) (T, error) {
	var v T
	err := json.NewDecoder(r.Body).Decode(&v)
	return v, err
}

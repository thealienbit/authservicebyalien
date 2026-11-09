package auth

import (
	"encoding/json"
	"net/http"

	"github.com/thealienbit/authservicebyalien/internal/signing"
)

type JWKSResponse struct {
	Keys []map[string]string `json:"keys"`
}

func JWKSHandler(keys []*signing.Key) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		resp := JWKSResponse{}

		for _, k := range keys {
			resp.Keys = append(resp.Keys, k.JWK())
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}


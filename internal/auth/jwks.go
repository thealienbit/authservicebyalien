package auth

import (
	"encoding/json"
	"net/http"
)

type JWKSResponse struct {
	Keys []map[string]string `json:"keys"`
}

func JWKSHandler(keys []*SigningKey) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		resp := JWKSResponse{}

		for _, k := range keys {
			resp.Keys = append(resp.Keys, publicKeyToJWKS(k))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}


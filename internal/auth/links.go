package auth

import "strings"

func tokenLink(baseURL, path, token string) string {
	return strings.TrimRight(baseURL, "/") + path + "?token=" + token
}

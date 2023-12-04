package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

type AuthMiddleware struct {
	Next http.Handler
}

func (m *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Next == nil {
		m.Next = http.DefaultServeMux
	}
	fmt.Println("auth...")

	Authorization := r.Header.Get("Authorization")
	fmt.Println(Authorization)
	if Authorization == "" {
		w.WriteHeader(http.StatusUnauthorized)
		errMsg, _ := json.Marshal(RequestError{
			Message: "未提供Key",
			Type:    NoKey,
		})
		fmt.Fprintf(w, "data: %s\n\n", errMsg)
		return
	}

	pattern, _ := regexp.Compile(`Bearer (.+)`)
	matches := pattern.FindStringSubmatch(Authorization)

	if len(matches) != 2 {
		errMsg, _ := json.Marshal(RequestError{
			Message: "未提供Key",
			Type:    NoKey,
		})
		fmt.Fprintf(w, "data: %s\n\n", errMsg)
		return
	}

	key := matches[1]
	fmt.Println("key", key)

	if key != "lzm" {
		w.WriteHeader(http.StatusUnauthorized)
		errMsg, _ := json.Marshal(RequestError{
			Message: "无效的Key",
			Type:    InvalidKey,
		})
		fmt.Fprintf(w, "data: %s\n\n", errMsg)
		return
	}

	m.Next.ServeHTTP(w, r)
}

type CorsMiddleware struct {
	Next http.Handler
}

func (c *CorsMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, X-Extra-Header, Content-Type, Accept, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")
	}

	if method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	c.Next.ServeHTTP(w, r)
}

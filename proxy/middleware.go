package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
)

type AuthMiddleware struct {
	Next http.Handler
}

type Result struct {
	Code   int
	Result struct {
		Message bool `json:"message"`
	}
}

func checkToken(url, token string) (Result, error) {
	client := &http.Client{}
	data := Result{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return data, fmt.Errorf("请求失败: %w", err)
	}

	defer resp.Body.Close()

	fmt.Println(resp.Body)
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&data)
	if err != nil {
		return data, fmt.Errorf("读取请求结果失败: %w", err)
	}

	log.Println(data)
	return data, nil
}

func (m *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Next == nil {
		m.Next = http.DefaultServeMux
	}
	fmt.Println("auth...", r.URL.Path)

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

	res, err := checkToken("http://localhost:5201/v1/token/check", Authorization)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !res.Result.Message {
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

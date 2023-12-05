package admin

import (
	"fmt"
	"log"
	"net/http"
	"os"

	jwt "github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
	Next http.Handler
}

func (m *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Next == nil {
		m.Next = http.DefaultServeMux
	}
	fmt.Println("auth...", r.URL.Path)

	if r.URL.Path == "/v1/login" {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		m.Next.ServeHTTP(w, r)
		return
	}

	secretkey, ok := os.LookupEnv("ADMIN_SECRETKEY")
	if !ok {
		log.Fatalf("请设置环境变量ADMIN_SECRETKEY")
	}
	jwt_token := r.Header.Get("Authorization")
	// fmt.Println(jwt_token)
	if jwt_token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(NewResponse(ERROR, Result{}.Message("缺少Token")))
		return
	}

	token, err := jwt.ParseWithClaims(jwt_token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretkey), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(NewResponse(ERROR, Result{}.Message(err)))
		return
	}

	if _, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		m.Next.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(NewResponse(ERROR, Result{}.Message("权限验证失败")))
	}
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

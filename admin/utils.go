package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	OK = iota
	ERROR
)

type Result struct{}

func (r Result) Message(msg any) map[string]any {
	return map[string]any{"message": msg}
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Code   int
	Result map[string]any
}

func NewResponse(code int, res map[string]any) []byte {
	res_ := Response{
		Code:   code,
		Result: res,
	}
	resp, _ := json.Marshal(res_)
	return resp
}

func (a *Admin) ShowAllDatabase() {
	database, _ := a.Client.ListDatabaseNames(context.Background(), bson.D{})
	fmt.Println(database)
}

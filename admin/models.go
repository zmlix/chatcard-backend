package admin

import "time"

type Quota int64
type Uid string
type Token string

type UserModel struct {
	Id     Uid
	Name   string
	Phone  string
	Tokens []Token
	Other  map[string]any
}

type TokenModel struct {
	Key        Token
	Number     Quota
	User       Uid
	CreateTime time.Duration
	UpdateTime time.Duration
	Models     []string
	Plugins    []string
	Disabled   bool
}

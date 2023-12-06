package admin

import "time"

type Quota int64
type Uid string
type Token string

type UserModel struct {
	Id     Uid            `bson:"id" json:"id" form:"id"`
	Name   string         `json:"name" form:"name" valid:"required,type(string)" bson:"name"`
	Phone  string         `json:"phone" form:"phone" valid:"matches(^1[3-9]{1}\\d{9}$)" bson:"phone"`
	Tokens []Token        `bson:"tokens"`
	Other  map[string]any `bson:"others"`
}

type TokenModel struct {
	Key        Token          `bson:"key"`
	Number     Quota          `bson:"number" valid:"required"`
	UserId     Uid            `bson:"user_id" valid:"required"`
	CreateTime time.Duration  `bson:"create_time"`
	UpdateTime time.Duration  `bson:"update_time"`
	Models     []string       `bson:"models"`
	Plugins    []string       `bson:"plugins"`
	Disabled   bool           `bson:"disabled"`
	Other      map[string]any `bson:"others"`
}

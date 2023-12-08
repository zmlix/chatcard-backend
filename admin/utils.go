package admin

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
)

const (
	OK = iota
	ERROR
)

const (
	UserTable  = "user"
	TokenTable = "token"
)

const (
	pageSize = 10
)

type Result struct{}

func (r Result) Message(msg any) map[string]any {
	return map[string]any{"message": msg}
}

func (r Result) Data(data any) map[string]any {
	return map[string]any{"data": data}
}

func (r Result) DataAndTotalPages(data any, totalPages int) map[string]any {
	return map[string]any{"data": data, "totalPages": totalPages}
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserCreate struct {
	Name  string `valid:"required,type(string)"`
	Phone string `valid:"matches(^1[3-9]{1}\\d{9}$)"`
}

type UserModify struct {
	UserId Uid            `json:"id"`
	Name   string         `json:"name"`
	Phone  string         `json:"phone"`
	Other  map[string]any `json:"other"`
}

type TokenCreate struct {
	UserId Uid   `json:"id"`
	Number Quota `json:"number"`
}

type TokenDelete struct {
	Key Token `json:"key"`
}

type TokenUpdateNumber struct{
	Key Token `json:"key" valid:"required"`
	Number Quota `json:"number" valid:"required"`
}

type TokenGetByKey struct{
	Key Token `json:"key"`
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

// DB
func (a *Admin) DBCreateUser(u *UserCreate) error {
	user := &UserModel{
		Id:    Uid(uuid.New().String()),
		Name:  u.Name,
		Phone: u.Phone,
	}

	collection := a.DBClient.Collection(UserTable)
	_, err := collection.InsertOne(context.TODO(), user)
	return err
}

func (a *Admin) DBCreateToken(t *TokenCreate) error {
	user := a.DBFindUserByID(t.UserId)
	if user == nil {
		return fmt.Errorf("用户不存在")
	}
	key, err := GenerateKeyByUserID(t.UserId)
	if err != nil {
		return err
	}
	token := &TokenModel{
		UserId:     t.UserId,
		Key:        Token(key),
		Number:     Quota(t.Number),
		CreateTime: time.Duration(time.Now().Unix()),
		UpdateTime: time.Duration(time.Now().Unix()),
		Disabled:   true,
	}

	_, err = a.DBClient.Collection(TokenTable).InsertOne(context.TODO(), token)
	if err != nil {
		return err
	}
	filter := bson.M{"id": user.Id}
	update := bson.D{bson.E{Key: "$push", Value: bson.D{{Key: "tokens", Value: token.Key}}}}
	_, err = a.DBClient.Collection(UserTable).UpdateOne(context.TODO(), filter, update)
	return err
}

func (a *Admin) DBCreate(data interface{}) error {
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("expected a struct, got %T", data)
	}
	fields := make(map[string]interface{})
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		fieldValue := value.Field(i).Interface()
		bsonTag := field.Tag.Get("bson")
		if bsonTag == "" {
			bsonTag = field.Name
		}
		fields[bsonTag] = fieldValue
	}

	collectionName := strings.ToLower(value.Type().Name())
	collectionName = strings.TrimSuffix(collectionName, "model")
	collection := a.DBClient.Collection(collectionName)
	if _, err := collection.InsertOne(context.TODO(), fields); err != nil {
		fmt.Println("Error storing data:", err)
		return err
	}
	return nil
}

func (a *Admin) DBFindUserByName(name string) *UserModel {
	user := &UserModel{}
	collection := a.DBClient.Collection(UserTable)
	filter := bson.M{"name": name}
	err := collection.FindOne(context.TODO(), filter).Decode(user)
	if err != nil {
		return nil
	}
	return user
}

func (a *Admin) DBFindUserByID(id Uid) *UserModel {
	user := &UserModel{}
	collection := a.DBClient.Collection(UserTable)
	filter := bson.M{"id": id}
	err := collection.FindOne(context.TODO(), filter).Decode(user)
	if err != nil {
		return nil
	}
	return user
}

func (a *Admin) DBFindTokenByKey(key Token) *TokenModel {
	token := &TokenModel{}
	collection := a.DBClient.Collection(TokenTable)
	filter := bson.M{"key": key}
	err := collection.FindOne(context.TODO(), filter).Decode(token)
	if err != nil {
		return nil
	}
	return token
}

func (a *Admin) DBFindAll(data interface{}) error {
	switch data := data.(type) {
	case *[]UserModel:
		collection := a.DBClient.Collection(UserTable)
		cursor, err := collection.Find(context.TODO(), bson.M{})
		if err != nil {
			return err
		}
		defer cursor.Close(context.TODO())
		for cursor.Next(context.TODO()) {
			var userModel UserModel
			if err := cursor.Decode(&userModel); err != nil {
				return err
			}
			*data = append(*data, userModel)
		}

	case *[]TokenModel:
		collection := a.DBClient.Collection(TokenTable)
		cursor, err := collection.Find(context.TODO(), bson.M{})
		if err != nil {
			return err
		}
		defer cursor.Close(context.TODO())
		for cursor.Next(context.TODO()) {
			var tokenModel TokenModel
			if err := cursor.Decode(&tokenModel); err != nil {
				return err
			}
			*data = append(*data, tokenModel)
		}

	default:
		return fmt.Errorf("不支持的数据类型: %T", data)
	}
	return nil
}

func (a *Admin) DBFindPage(data interface{}, page int) (int, error) {
	if page < 1 {
		page = 1
	}
	switch data := data.(type) {
	case *[]UserModel:
		collection := a.DBClient.Collection(UserTable)
		count, err := collection.CountDocuments(context.TODO(), bson.M{})
		if err != nil {
			return 0, err
		}
		totalPages := int(math.Ceil(float64(count) / float64(pageSize)))
		if totalPages == 0 {
			return 0, nil
		}
		if page > totalPages {
			page = totalPages
		}
		offset := pageSize * (page - 1)
		cursor, err := collection.Find(
			context.TODO(),
			bson.M{},
			options.Find().SetSkip(int64(offset)).SetLimit(int64(pageSize)),
		)
		if err != nil {
			return 0, err
		}
		defer cursor.Close(context.TODO())
		for cursor.Next(context.TODO()) {
			var userModel UserModel
			if err := cursor.Decode(&userModel); err != nil {
				return 0, err
			}
			*data = append(*data, userModel)
		}
		return totalPages, nil

	case *[]TokenModel:
		collection := a.DBClient.Collection(TokenTable)
		count, err := collection.CountDocuments(context.TODO(), bson.M{})
		if err != nil {
			return 0, err
		}
		totalPages := int(math.Ceil(float64(count) / float64(pageSize)))
		if totalPages == 0 {
			return 0, nil
		}
		if page > totalPages {
			page = totalPages
		}
		offset := pageSize * (page - 1)
		cursor, err := collection.Find(
			context.TODO(),
			bson.M{},
			options.Find().SetSkip(int64(offset)).SetLimit(int64(pageSize)),
		)
		if err != nil {
			return 0, err
		}
		defer cursor.Close(context.TODO())
		for cursor.Next(context.TODO()) {
			var tokenModel TokenModel
			if err := cursor.Decode(&tokenModel); err != nil {
				return 0, err
			}
			*data = append(*data, tokenModel)
		}
		return totalPages, nil
	default:
		return 0, fmt.Errorf("不支持的数据类型: %T", data)
	}
}

func (a *Admin) DBUpdateUser(user *UserModify) error {
	userfound := a.DBFindUserByID(user.UserId)
	if userfound == nil {
		return fmt.Errorf("用户不存在")
	}
	collection := a.DBClient.Collection(UserTable)
	filter := bson.M{"id": user.UserId}
	update := bson.D{}
	if user.Name != "" {
		if u := a.DBFindUserByName(user.Name); u == nil || userfound.Id == user.UserId {
			update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "name", Value: user.Name}}})
		} else {
			return fmt.Errorf("用户名已存在")
		}
	}
	if user.Phone != "" {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "phone", Value: user.Phone}}})
	}
	if len(user.Other) > 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "others", Value: user.Other}}})
	}
	_, err := collection.UpdateOne(context.TODO(), filter, update)
	return err
}

func (a *Admin) DBDeleteToken(token *TokenDelete) error {
	tokenfound := a.DBFindTokenByKey(token.Key)
	if tokenfound == nil {
		return fmt.Errorf("Token不存在")
	}
	collection := a.DBClient.Collection(TokenTable)
	filter := bson.M{"key": token.Key}
	if _, err := collection.DeleteOne(context.TODO(), filter); err != nil {
		return err
	}
	userfound := a.DBFindUserByID(tokenfound.UserId)
	if userfound == nil {
		return fmt.Errorf("该token无主")
	}
	user_collection := a.DBClient.Collection(UserTable)
	user_filter := bson.M{"id": tokenfound.UserId}
	update := bson.D{bson.E{Key: "$pull", Value: bson.D{{Key: "tokens", Value: token.Key}}}}
	_, err := user_collection.UpdateOne(context.TODO(), user_filter, update)
	return err
}

func (a *Admin) DBFindTokenByUserId(userId Uid) *TokenModel {
	collection := a.DBClient.Collection(TokenTable)
	filter := bson.M{"user_id": userId}
	var token *TokenModel
	err := collection.FindOne(context.TODO(), filter).Decode(&token)
	if err != nil {
		return nil
	}
	return token
}

func (a *Admin) DBUpdateTokenNumber(token *TokenUpdateNumber) error {
	tokenfound := a.DBFindTokenByKey(token.Key)
	if tokenfound == nil {
		return fmt.Errorf("token不存在")
	}
	if (tokenfound.Number + token.Number < 0) && token.Number < 0{
		return fmt.Errorf("token Number 不足")
	}
	collection := a.DBClient.Collection(TokenTable)
	filter := bson.M{"key": token.Key}
	update := bson.D{{Key: "$inc", Value: bson.D{{Key: "number", Value: token.Number}}}, 
	{Key: "$set", Value: bson.D{{Key: "update_time", Value: time.Now().Unix()}}}}
    if _, err := collection.UpdateOne(context.TODO(), filter, update); err != nil {
        return err
    }
    return nil
}

// 生成用户key
func generateKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(key), nil
}

func GenerateKeyByUserID(userID Uid) (string, error) {
	secretKey, err := generateKey()
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, []byte(secretKey))
	_, err = h.Write([]byte(userID))
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(h.Sum(nil)), nil
}

func CheckRequestMethod(method string, methods []string) bool {
	return slices.Contains(methods, method)
}

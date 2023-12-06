package admin

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strings"
	"time"

	mango "go.mongodb.org/mongo-driver/mongo"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

const (
	OK = iota
	ERROR
)

var DBname = "gpt_server"
var pageSize = 10

type Result struct{}

func (r Result) Message(msg any) map[string]any {
	return map[string]any{"message": msg}
}

func (r Result) Data(data any) map[string]any{
	return map[string]any{"data": data}
}

func (r Result) DataAndTotalPages(data any, totalPages int) map[string]any{
	return map[string]any{"data": data, "totalPages": totalPages}
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
	database, _ := a.DBClient.ListDatabaseNames(context.Background(), bson.D{})
	fmt.Println(database)
}


// DB
func (a *Admin) DBCreateUserAndToken(user UserModel) error {
	ctx := context.Background()
	writeconcern := writeconcern.Majority()
	session, err := a.DBClient.StartSession()
	if err != nil {
		Error(err.Error() + "CreateUserAndToken")
		Fatal(err)
		return err
	}
	defer session.EndSession(ctx)

	err = session.StartTransaction(
		options.Transaction().
			SetWriteConcern(writeconcern),
	)
	if err != nil {
		Fatal(err)
		return err
	}

	userid := uuid.New().String()
	user.Id = Uid(userid)
	keyValue, err := GenerateKeyByUserID(user.Id)
	if  err != nil{
		Fatal(err)
		return errors.New("生成Key失败")
	}
	token := TokenModel{
		UserId: user.Id,
		Key: Token(keyValue),
		Number: Quota(0),
		CreateTime: time.Duration(time.Now().Unix()),
		Disabled: true,
	}
	user.Tokens = append(user.Tokens, token.Key)
	err = a.DBCreate(token)
	if err != nil {
		_ = session.AbortTransaction(ctx)
		Fatal(err)
		return err
	}
	if err = a.DBCreate(user); err != nil {
		_ = session.AbortTransaction(ctx)
		Fatal(err)
		return err
	}
	//commit
	err = session.CommitTransaction(ctx)
	if err != nil {
		Error(err.Error() + "CreateUserAndToken")
		Fatal(err)
		return err
	}
	return nil
}

func (a *Admin) DBCreate(data interface{}) error {
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("Expected a struct, got %T", data)
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
	collection := a.DBClient.Database(DBname).Collection(collectionName)
	if _, err := collection.InsertOne(context.TODO(), fields); err != nil {
		fmt.Println("Error storing data:", err)
		return err
	}
	return nil
}

func (a *Admin) DBFindUserByName(name string) (UserModel, error) {
	user := UserModel{}
	collection := a.DBClient.Database(DBname).Collection("user")
	filter := bson.M{"name": name}
	err := collection.FindOne(context.TODO(), filter).Decode(&user)
	return user, err
}

func (a *Admin) DBFindUserByID(id Uid) (UserModel, error) {
	user := UserModel{}
	collection := a.DBClient.Database(DBname).Collection("user")
	filter := bson.M{"id": id}
	err := collection.FindOne(context.TODO(), filter).Decode(&user)
	return user, err
}

func (a *Admin) DBFindTokenByKey(key Token) (TokenModel, error) {
	token := TokenModel{}
	collection := a.DBClient.Database(DBname).Collection("user")
	filter := bson.M{"key": key}
	err := collection.FindOne(context.TODO(), filter).Decode(&token)
	return token, err
}

func (a *Admin) DBFindAll(data interface{}) error {
	switch data := data.(type) {
	case *[]UserModel:
		collection := a.DBClient.Database(DBname).Collection("user")
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
		collection := a.DBClient.Database(DBname).Collection("token")
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
		collection := a.DBClient.Database(DBname).Collection("user")
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
		collection := a.DBClient.Database(DBname).Collection("token")
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

func (a *Admin) DBUpdate(data interface{}) error {
	switch v := data.(type){
	case UserModel:
		return a.DBUpdateUser(v)
	case TokenModel:
		return a.DBUpdateToken(v)
	default:
		return fmt.Errorf("不支持的数据类型: %T", data)
	}
}

func (a *Admin) DBUpdateUser(user UserModel) error {
	userfound, err := a.DBFindUserByID(user.Id);
	if  err != nil {
		return err
	}
	collection := a.DBClient.Database(DBname).Collection("user")
	filter := bson.M{"id": user.Id}
	update := bson.D{}
	if user.Name != "" {
		if _, err := a.DBFindUserByName(user.Name); err == mango.ErrNoDocuments || userfound.Name == user.Name {
			update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "name", Value: user.Name}}})
		}else{
			return fmt.Errorf("用户名已存在")
		}
	}
	if user.Phone != "" {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "phone", Value: user.Phone}}})
	}
	if len(user.Tokens) > 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "tokens", Value: user.Tokens}}})
	}
	if len(user.Other) > 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "others", Value: user.Other}}})
	}
	_, err = collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (a *Admin) DBUpdateToken(token TokenModel) error {
	if _, err := a.DBFindTokenByKey(token.Key); err != nil {
		return nil
	}
	collection := a.DBClient.Database(DBname).Collection("token")
	filter := bson.M{"key": token.Key}
	update := bson.D{}
	if token.Number != 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "number", Value: token.Number}}})
	}
	if token.UserId != "" {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "user_id", Value: token.UserId}}})
	}
	if token.CreateTime != 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "create_time", Value: token.CreateTime}}})
	}
	if token.UpdateTime != 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "update_time", Value: token.UpdateTime}}})
	}
	if len(token.Models) > 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "models", Value: token.Models}}})
	}
	if len(token.Plugins) > 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "plugins", Value: token.Plugins}}})
	}
	if token.Disabled {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "disabled", Value: token.Disabled}}})
	}
	if len(token.Other) > 0 {
		update = append(update, bson.E{Key: "$set", Value: bson.D{{Key: "others", Value: token.Other}}})
	}
	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}
	return nil
}

//生成用户key
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

func CheckRequestMethod(w http.ResponseWriter, r *http.Request, method string) error {
	if r.Method == method{
		return nil
	}else{
		w.Write(NewResponse(ERROR, Result{}.Message("请求方法错误！")))
		return errors.New("请求方法错误！")
	}
}


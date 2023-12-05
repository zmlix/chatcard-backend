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
	"net/http"
	"reflect"
	"strings"
	"time"

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

type Result struct{}

func (r Result) Message(msg any) map[string]any {
	return map[string]any{"message": msg}
}

func (r Result) Data(data any) map[string]any{
	return map[string]any{"data": data}
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

func (a *Admin) DBFindALL(data interface{}) ([]interface{}, error) {
	var result []interface{}
	switch data.(type) {
	case UserModel:
		collection := a.DBClient.Database(DBname).Collection("user")
		cursor, err := collection.Find(context.TODO(), bson.M{})
		if err != nil {
			fmt.Println("sad")
			return nil, err
		}
		defer cursor.Close(context.TODO())
		for cursor.Next(context.TODO()) {
			var userModel UserModel
			err := cursor.Decode(&userModel)
			if err != nil {
				return nil, err
			}
			result = append(result, userModel)
		}
		if err := cursor.All(context.TODO(), &result); err != nil {
			return nil, err
		}

	case TokenModel:
		collection := a.DBClient.Database(DBname).Collection("token")
		cursor, err := collection.Find(context.TODO(), bson.M{})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(context.TODO())
		for cursor.Next(context.TODO()) {
			var tokenModel TokenModel
			err := cursor.Decode(&tokenModel)
			if err != nil {
				return nil, err
			}
			result = append(result, tokenModel)
		}

	default:
		return nil, fmt.Errorf("unsupported data type: %T", data)
	}
	return result, nil
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


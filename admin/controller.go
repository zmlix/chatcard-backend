package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/golang-jwt/jwt/v5"
)

func (a *Admin) Login(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	userLogin := UserLogin{}
	err := dec.Decode(&userLogin)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username, ok := os.LookupEnv("ADMIN_USERNAME")
	if !ok {
		log.Fatalf("请设置环境变量ADMIN_USERNAME")
	}
	password, ok := os.LookupEnv("ADMIN_PASSWORD")
	if !ok {
		log.Fatalf("请设置环境变量ADMIN_PASSWORD")
	}
	secretkey, ok := os.LookupEnv("ADMIN_SECRETKEY")
	if !ok {
		log.Fatalf("请设置环境变量ADMIN_SECRETKEY")
	}

	if username == userLogin.Username && password == userLogin.Password {
		t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		})
		token, err := t.SignedString([]byte(secretkey))
		if err != nil {
			log.Printf("签发JWT-Token失败 %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Add("Token", token)
		w.Write(NewResponse(OK, Result{}.Message("登陆成功")))
		return
	}
	w.Write(NewResponse(ERROR, Result{}.Message("登陆失败")))
}

func (a *Admin) CreateUser(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(r.Method, []string{http.MethodPost}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user := &UserCreate{}
	json.NewDecoder(r.Body).Decode(user)
	_, err := govalidator.ValidateStruct(user)
	if err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("检验不通过："+err.Error())))
		return
	}

	if userfound := a.DBFindUserByName(user.Name); userfound != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("用户"+userfound.Name+"已存在")))
		return
	}

	if err := a.DBCreateUser(user); err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("创建失败："+err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.Message("创建成功!")))
}

func (a *Admin) GetUserList(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(r.Method, []string{http.MethodGet}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var err error
	var page int
	if r.URL.Query().Has("page") {
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			w.Write(NewResponse(ERROR, Result{}.Message("page值非法")))
		}
	} else {
		page = 1
	}
	userList := []UserModel{}
	// err := a.DBFindAll(&userList)
	totalPages, err := a.DBFindPage(&userList, page)
	if err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("查询失败："+err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.DataAndTotalPages(userList, totalPages)))
}

func (a *Admin) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(r.Method, []string{http.MethodPost}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user := &UserModify{}
	json.NewDecoder(r.Body).Decode(user)
	if user.UserId == "" {
		w.Write(NewResponse(ERROR, Result{}.Message("用户ID不能为空")))
		return
	}
	if err := a.DBUpdateUser(user); err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("更新失败："+err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.Message("更新成功")))
}


func (a *Admin) CreateToken(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(r.Method, []string{http.MethodPost}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := &TokenCreate{}
	json.NewDecoder(r.Body).Decode(token)
	if token.UserId == "" {
		w.Write(NewResponse(ERROR, Result{}.Message("用户ID不能为空")))
		return
	}
	if err := a.DBCreateToken(token); err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("创建失败："+err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.Message("创建成功!")))
}

func (a *Admin) DeleteToken(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(r.Method, []string{http.MethodPost}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := &TokenDelete{}
	json.NewDecoder(r.Body).Decode(token)
	if token.Key == "" {
		w.Write(NewResponse(ERROR, Result{}.Message("key值不能为空")))
		return
	}
	if err := a.DBDeleteToken(token); err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("删除失败："+err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.Message("删除成功!")))
}

func (a *Admin) GetTokenList(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(r.Method, []string{http.MethodGet}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var err error
	var page int
	if r.URL.Query().Has("page") {
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			w.Write(NewResponse(ERROR, Result{}.Message("page值非法")))
		}
	} else {
		page = 1
	}
	tokenList := []TokenModel{}
	totalPages, err := a.DBFindPage(&tokenList, page)
	if err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("查询失败："+err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.DataAndTotalPages(tokenList, totalPages)))
}

func (a *Admin) UpdateTokenNumber(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(r.Method, []string{http.MethodPost}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := TokenUpdateNumber{}
	json.NewDecoder(r.Body).Decode(&token)
	_, err := govalidator.ValidateStruct(token)
	if err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("数据检验不通过："+err.Error())))
		return
	}
	if err := a.DBUpdateTokenNumber(&token); err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("UpdateTokenNumber失败: "+ err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.Message("操作成功")))
}

func (a *Admin) GetTokenByKey(w http.ResponseWriter, r *http.Request){
	if !CheckRequestMethod(r.Method, []string{http.MethodPost}) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	key := TokenGetByKey{}
	json.NewDecoder(r.Body).Decode(&key)
	if key.Key == "" {
		w.Write(NewResponse(ERROR, Result{}.Message("key值不能为空")))
		return
	}

	token := a.DBFindTokenByKey(key.Key)
	if token == nil {
		w.Write(NewResponse(ERROR, Result{}.Message("查询为空")))
		return
	}
	w.Write(NewResponse(OK, Result{}.Data(token)))
}
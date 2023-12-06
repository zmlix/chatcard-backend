package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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
	if err := CheckRequestMethod(w, r, "POST"); err != nil {
		return
	}
	user := UserModel{}
	json.NewDecoder(r.Body).Decode(&user)
	_, err := govalidator.ValidateStruct(user)
	if err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("检验不通过：" + err.Error())))
		return
	}
	userfound, _ := a.DBFindUserByName(user.Name)
	if  userfound.Id != "" {
		w.Write(NewResponse(ERROR, Result{}.Message("用户名已存在")))
		return
	}
	if err := a.DBCreateUserAndToken(user); err != nil {
		w.Write(NewResponse(ERROR, Result{}.Message("创建失败：" + err.Error())))
		return
	}
	w.Write(NewResponse(OK, Result{}.Message("创建成功!")))
}

func (a *Admin) GetUserList(w http.ResponseWriter, r *http.Request){
	if err := CheckRequestMethod(w, r, "GET"); err != nil {
		return
	}
	userList := []UserModel{}
	err := a.DBFindAll(&userList)
	if err != nil{
		w.Write(NewResponse(ERROR, Result{}.Message("查询失败：" + err.Error())))
	}
	w.Write(NewResponse(OK, Result{}.Data(userList)))
}


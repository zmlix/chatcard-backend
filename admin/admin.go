package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Admin struct {
	Client *mongo.Client
}

func New() *Admin {
	uri, ok := os.LookupEnv("MONGODB_URI")
	if !ok {
		log.Fatalf("请设置环境变量MONGODB_URI")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	return &Admin{
		Client: client,
	}
}

func Run(addr string, admin *Admin) {
	fmt.Printf("Admin running in %s ...\n", addr)

	if admin == nil {
		admin = New()
	}

	handler := &CorsMiddleware{
		Next: &AuthMiddleware{
			Next: admin,
		},
	}

	server := http.Server{
		Addr:    addr,
		Handler: handler,
	}

	server.ListenAndServe()
}

func (a *Admin) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.URL.Path {
	case "/v1/login":
		a.Login(w, r)
	case "/v1/create/user":
		a.CreateUser(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

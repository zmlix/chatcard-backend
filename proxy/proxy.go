package proxy

import (
	"log"
	"net/http"
	"os"
	"regexp"

	openai "github.com/sashabaranov/go-openai"
)

type Proxy struct {
	Url    string
	Key    string
	Client *openai.Client
}

func New(key string) *Proxy {

	conf := openai.DefaultConfig(key)
	client := openai.NewClientWithConfig(conf)

	return &Proxy{
		Url:    conf.BaseURL, // https://api.openai.com/v1
		Key:    key,
		Client: client,
	}
}

func NewWithUrl(key string, url string) *Proxy {

	conf := openai.DefaultConfig(key)
	if url != "" {
		conf.BaseURL = url
	}
	client := openai.NewClientWithConfig(conf)

	return &Proxy{
		Url:    url,
		Key:    key,
		Client: client,
	}
}

func Run(addr string, proxy *Proxy) {

	key, ok := os.LookupEnv("OPENAI_KEY")
	if !ok {
		log.Fatalf("请设置环境变量OPENAI_KEY")
	}

	if proxy == nil {
		proxy = New(key)
	}

	handler := &CorsMiddleware{
		Next: &AuthMiddleware{
			Next: proxy,
		},
	}

	server := http.Server{
		Addr:    addr,
		Handler: handler,
	}

	server.ListenAndServe()
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	pattern, _ := regexp.Compile(`/v1(/.+)`)
	matches := pattern.FindStringSubmatch(r.URL.Path)

	if len(matches) > 0 {
		url := matches[1]
		switch url {
		case "/chat/completions":
			p.ChatCompletion(w, r)
		case "/models":
			p.ModelList(w, r)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

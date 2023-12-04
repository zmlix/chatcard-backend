package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	openai "github.com/sashabaranov/go-openai"
)

func (p *Proxy) ChatCompletion(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost && r.Method != http.MethodOptions {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	client := p.Client
	ctx := context.Background()
	dec := json.NewDecoder(r.Body)

	req := openai.ChatCompletionRequest{}
	err := dec.Decode(&req)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Printf("%+v\n", req)

	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Printf("ChatCompletionStream error: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		APIError, ok := err.(*openai.APIError)
		if ok {
			errMsg, _ := json.Marshal(APIError)
			fmt.Fprintf(w, "data: %s\n\n", errMsg)
		}
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	fmt.Printf("Stream response: ")
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream finished")
			return
		}

		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			return
		}

		fmt.Println(response)
		data, _ := json.Marshal(response)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}

}

func (p *Proxy) ModelList(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	modelList, _ := p.Client.ListModels(context.Background())
	data, _ := json.Marshal(modelList)
	w.Write(data)
}

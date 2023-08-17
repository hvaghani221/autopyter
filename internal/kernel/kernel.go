package kernel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var host, token string

func InitKernel(kernelHost, kernelToken string) {
	host = kernelHost
	token = kernelToken
}

type Kernel struct {
	ID           string `json:"id"`
	closeChan    chan struct{}
	codeRequests map[string]executeRequest
	conn         *websocket.Conn
	messageChan  chan Message
	sessionId    string
}

func CreateKernel() (*Kernel, error) {
	if host == "" {
		return nil, fmt.Errorf("host is not initialized")
	}
	header := http.Header{}
	if token != "" {
		header.Add("Authorization", "token "+token)
	}

	kernelUrl := url.URL{Scheme: "http", Host: host, Path: "/api/kernels"}

	reqBody := map[string]any{
		"name": "python3",
	}
	content, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", kernelUrl.String(), bytes.NewBuffer(content))
	if err != nil {
		return nil, err
	}

	req.Header = header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("resp: %s, status code: %s", string(respBytes), resp.Status)
	}
	var kernel Kernel
	if err := json.Unmarshal(respBytes, &kernel); err != nil {
		return nil, err
	}

	kernel.sessionId = uuid.New().String()
	kernel.codeRequests = make(map[string]executeRequest)

	kernel.closeChan = make(chan struct{})
	if err = kernel.listenToKernel(); err != nil {
		return nil, err
	}

	go kernel.handleKernelMessages()
	return &kernel, nil
}

func (k *Kernel) ExecuteCode(code string) (*ResultMessage, *ExceptionMessage, error) {
	if len(strings.TrimSpace(code)) == 0 {
		return &ResultMessage{}, nil, nil
	}
	req := executeRequest{
		code:          code,
		messageId:     uuid.New().String(),
		resultChan:    make(chan ResultMessage, 1),
		exceptionChan: make(chan ExceptionMessage, 1),
	}
	k.codeRequests[req.messageId] = req
	defer func() {
		delete(k.codeRequests, req.messageId)
	}()

	if err := k.sendExecuteRequest(req); err != nil {
		return nil, nil, err
	}
	select {
	case result := <-req.resultChan:
		return &result, nil, nil
	case exception := <-req.exceptionChan:
		return nil, &exception, nil
	}
}

func (k *Kernel) listenToKernel() error {
	header := http.Header{}
	if token != "" {
		header.Add("Authorization", "token "+token)
	}
	u := url.URL{Scheme: "ws", Host: host, Path: fmt.Sprintf("api/kernels/%s/channels", k.ID)}

	u.Query().Add("session_id", k.sessionId)
	client, resp, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		_ = err
		log.Println("connection ", err)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(resp.Status, string(body))
	}

	ch := make(chan Message)
	k.messageChan = ch
	k.conn = client

	go func() {
		for {
			select {
			case <-k.closeChan:
				client.Close()
				return
			default:
				var op Message
				err = client.ReadJSON(&op)
				if err != nil {
					log.Fatal(err)
				}
				ch <- op
			}
		}
	}()
	return nil
}

func (k *Kernel) Close() {
	log.Println("Closing kernel", k.ID)
	close(k.closeChan)
	if err := k.deleteKernel(); err != nil {
		log.Fatal(err)
	}
}

func (k *Kernel) handleKernelMessages() {
	var stream *StreamMessage
	for {
		select {
		case <-k.closeChan:
			return
		case msg := <-k.messageChan:
			request, ok := k.codeRequests[msg.ParentHeader.MsgID]
			if !ok {
				continue
			}
			switch msg.MsgType {
			case Stream:
				stream = parseStreamMessage(msg.Content)
			case ExecuteResult, DisplayData:
				res := parseResultMessage(msg.Content)
				if stream != nil {
					res.Stream = stream
				}
				request.resultChan <- res
			case ExecuteReply:
				res := ResultMessage{}
				if stream != nil {
					res.Stream = stream
				}
				request.resultChan <- res
			case ExecuteError:
				request.exceptionChan <- parseErrorMessage(msg.Content)
			}
		}
	}
}

func (k *Kernel) sendExecuteRequest(req executeRequest) error {
	msg := map[string]interface{}{
		"header": map[string]interface{}{
			"msg_id":   req.messageId,
			"username": "",
			"session":  k.sessionId,
			"msg_type": "execute_request",
			"version":  "5.2",
			"date":     time.Now().Format(time.RFC3339),
		},
		"content": map[string]interface{}{
			"code":             req.code,
			"silent":           false,
			"store_history":    false,
			"allow_stdin":      false,
			"stop_on_error":    true,
			"user_expressions": map[string]interface{}{},
		},
		"parent_header": map[string]interface{}{},
		"metadata":      map[string]interface{}{},
		"buffers":       []interface{}{},
	}

	err := k.conn.WriteJSON(msg)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kernel) deleteKernel() error {
	header := http.Header{}
	if token != "" {
		header.Add("Authorization", "token "+token)
	}

	kernelUrl := url.URL{Scheme: "http", Host: "localhost:8888", Path: "/api/kernels/" + k.ID}

	req, err := http.NewRequest("DELETE", kernelUrl.String(), nil)
	if err != nil {
		return err
	}

	req.Header = header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

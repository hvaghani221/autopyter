package kernel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	host, token string
	debug       bool
)

func InitKernel(kernelHost, kernelToken string, debugFlag bool) {
	host = kernelHost
	token = kernelToken
	debug = debugFlag

	if debug {
		_ = os.RemoveAll("logs/")
		_ = os.MkdirAll("logs", 0o777)
	}
}

type Kernel struct {
	ID           string
	closeChan    chan struct{}
	codeRequests map[string]*executeRequest
	conn         *websocket.Conn
	messageChan  chan Message
	sessionId    string
	file         *os.File
	mu           sync.Mutex
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

	if debug {
		kernel.file, err = os.Create("logs/" + kernel.ID + ".log")
		if err != nil {
			return nil, err
		}
	}

	kernel.mu = sync.Mutex{}

	kernel.sessionId = uuid.New().String()
	kernel.codeRequests = make(map[string]*executeRequest)

	kernel.closeChan = make(chan struct{})
	if err = kernel.listenToKernel(); err != nil {
		return nil, err
	}
	go kernel.handleKernelMessages()
	return &kernel, nil
}

func (k *Kernel) ExecuteCode(code string) ([]ResultMessage, []ExceptionMessage, error) {
	if len(strings.TrimSpace(code)) == 0 {
		return []ResultMessage{}, nil, nil
	}
	req := executeRequest{
		code:      code,
		messageId: uuid.New().String(),
		result:    make([]ResultMessage, 0, 1),
		exception: make([]ExceptionMessage, 0, 1),
		doneChan:  make(chan struct{}),
	}
	k.mu.Lock()
	k.codeRequests[req.messageId] = &req
	k.mu.Unlock()

	if err := k.sendExecuteRequest(req); err != nil {
		return nil, nil, err
	}
	<-req.doneChan

	return req.result, req.exception, nil
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
				return
			default:
				var op Message
				err = client.ReadJSON(&op)
				k.log("Message type: %s ParentHeader: %s\n", op.MsgType, op.ParentHeader)
				content, _ := json.Marshal(op.Content)
				k.log("Content: %s\n", string(content))
				k.log("--------------------\n\n")
				select {
				case <-k.closeChan:
					return
				default:
					if err != nil {
						log.Println(err)
						continue
					}
					ch <- op
				}
			}
		}
	}()
	return nil
}

func (k *Kernel) Close() {
	defer k.conn.Close()
	if debug {
		defer k.file.Close()
	}
	if err := k.deleteKernel(); err != nil {
		log.Println(err)
	}
	close(k.closeChan)
}

func (k *Kernel) handleKernelMessages() {
	var stream *StreamMessage
	for {
		select {
		case <-k.closeChan:
			return
		case msg := <-k.messageChan:
			k.mu.Lock()
			request, ok := k.codeRequests[msg.ParentHeader.MsgID]
			k.mu.Unlock()
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
				stream = nil
				k.mu.Lock()
				request.result = append(request.result, res)
				k.mu.Unlock()
			case ExecuteReply:
				res := ResultMessage{}
				if stream != nil {
					res.Stream = stream
				}
				stream = nil
				k.mu.Lock()
				request.result = append(request.result, res)
				k.mu.Unlock()
			case ExecuteError:
				k.mu.Lock()
				request.exception = append(request.exception, parseErrorMessage(msg.Content))
				k.mu.Unlock()
			case Status:
				if msg.Content["execution_state"] == "idle" {
					request.doneChan <- struct{}{}
				}
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

	kernelUrl := url.URL{Scheme: "http", Host: host, Path: "/api/kernels/" + k.ID}

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

func (k *Kernel) log(format string, a ...any) {
	if debug {
		fmt.Fprintf(k.file, format, a...)
	}
}

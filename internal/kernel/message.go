package kernel

import "time"

type MessageType string

const (
	ExecuteRequest MessageType = "execute_request"
	ExecuteResult  MessageType = "execute_result"
	ExecuteError   MessageType = "error"
	ExecuteReply   MessageType = "execute_reply"
	Stream         MessageType = "stream"
	DisplayData    MessageType = "display_data"
)

type Message struct {
	Buffers      []interface{}  `json:"buffers"`
	Channel      string         `json:"channel"`
	Content      map[string]any `json:"content"`
	Header       Header         `json:"header"`
	Metadata     struct{}       `json:"metadata"`
	MsgID        string         `json:"msg_id"`
	MsgType      MessageType    `json:"msg_type"`
	ParentHeader Header         `json:"parent_header"`
}

type Header struct {
	Date     time.Time `json:"date"`
	MsgID    string    `json:"msg_id"`
	MsgType  string    `json:"msg_type"`
	Session  string    `json:"session"`
	Username string    `json:"username"`
	Version  string    `json:"version"`
}

type executeRequest struct {
	code          string
	messageId     string
	resultChan    chan ResultMessage
	exceptionChan chan ExceptionMessage
}

type ExceptionMessage struct {
	EName  string `json:"ename"`
	EValue string `json:"evalue"`
}

type ResultMessage struct {
	Data     map[string]any
	MetaData map[string]any
	Stream   *StreamMessage
}

type StreamMessage struct {
	Name string
	Text string
}

func parseErrorMessage(msg map[string]any) ExceptionMessage {
	return ExceptionMessage{
		EName:  msg["ename"].(string),
		EValue: msg["evalue"].(string),
	}
}

func parseResultMessage(msg map[string]any) ResultMessage {
	return ResultMessage{
		Data:     msg["data"].(map[string]any),
		MetaData: msg["metadata"].(map[string]any),
	}
}

func parseStreamMessage(msg map[string]any) *StreamMessage {
	return &StreamMessage{
		Name: msg["name"].(string),
		Text: msg["text"].(string),
	}
}

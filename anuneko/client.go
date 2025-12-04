package anuneko

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ChatAPIUrl      = "https://anuneko.com/api/v1/chat"
	StreamAPIUrl    = "https://anuneko.com/api/v1/msg/%s/stream"
	SelectChoiceUrl = "https://anuneko.com/api/v1/msg/select-choice"
	SelectModelUrl  = "https://anuneko.com/api/v1/user/select_model"
)

type AnuNekoClient struct {
	HttpClient   *http.Client
	UserSessions map[string]string // userID -> chatID
	UserModels   map[string]string // userID -> model
	Token        string
	DeviceId     string
	Cookie       string
}

func NewClient(Token string, Cookie string, DeviceId string) *AnuNekoClient {
	return &AnuNekoClient{
		HttpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		UserSessions: make(map[string]string),
		UserModels:   make(map[string]string),
		Token:        Token,
		Cookie:       Cookie,
		DeviceId:     DeviceId,
	}
}

func (c *AnuNekoClient) buildHeaders() http.Header {
	token := c.Token
	cookie := c.Cookie
	deviceId := c.DeviceId

	h := http.Header{}
	h.Set("accept", "*/*")
	h.Set("content-type", "application/json")
	h.Set("origin", "https://anuneko.com")
	h.Set("referer", "https://anuneko.com/")
	h.Set("user-agent", "Mozilla/5.0")
	h.Set("x-app_id", "com.anuttacon.neko")
	h.Set("x-client_type", "4")
	h.Set("x-device_id", deviceId)
	h.Set("x-token", token)

	if cookie != "" {
		h.Set("Cookie", cookie)
	}
	return h
}

func (c *AnuNekoClient) GetSessionId(uid int64) (string, bool) {
	userID := strconv.FormatInt(uid, 10)
	sid, ok := c.UserSessions[userID]
	return sid, ok
}

func (c *AnuNekoClient) CreateNewSession(uid int64) (string, error) {
	userID := strconv.FormatInt(uid, 10)
	model := c.UserModels[userID]
	if model == "" {
		model = "Orange Cat"
	}

	body := map[string]any{"model": model}
	bz, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", ChatAPIUrl, bytes.NewReader(bz))
	req.Header = c.buildHeaders()

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var j map[string]any
	json.NewDecoder(resp.Body).Decode(&j)

	chatID, _ := j["chat_id"].(string)
	if chatID == "" {
		chatID, _ = j["id"].(string)
	}

	if chatID == "" {
		return "", errors.New("no chat_id")
	}

	c.UserSessions[userID] = chatID
	c.SwitchModel(uid, chatID, model)

	return chatID, nil
}

func (c *AnuNekoClient) SwitchModel(uid int64, chatID, model string) bool {
	userID := strconv.FormatInt(uid, 10)
	body := map[string]any{
		"chat_id": chatID,
		"model":   model,
	}

	bz, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", SelectModelUrl, bytes.NewReader(bz))
	req.Header = c.buildHeaders()

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		c.UserModels[userID] = model
		return true
	}
	return false
}
func (c *AnuNekoClient) SendChoice(msgID string) {
	body := map[string]any{
		"msg_id":     msgID,
		"choice_idx": 0,
	}
	bz, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", SelectChoiceUrl, bytes.NewReader(bz))
	req.Header = c.buildHeaders()

	c.HttpClient.Do(req) // 忽略错误
}

func (c *AnuNekoClient) StreamReply(sessionID, text string) (string, error) {
	headers := http.Header{}
	token := c.Token

	headers.Set("x-token", token)
	headers.Set("Content-Type", "text/plain")

	if ck := c.Cookie; ck != "" {
		headers.Set("Cookie", ck)
	}

	reqBody := map[string]any{
		"contents": []string{text},
	}
	bz, _ := json.Marshal(reqBody)

	url := fmt.Sprintf(StreamAPIUrl, sessionID)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(bz))
	req.Header = headers

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	reader := bufio.NewScanner(resp.Body)

	var result strings.Builder
	var currentMsgID string

	for reader.Scan() {
		line := reader.Text()
		if line == "" {
			continue
		}

		// 非 data: 前缀 = 可能是错误
		if !strings.HasPrefix(line, "data: ") {
			var errJSON map[string]any
			if json.Unmarshal([]byte(line), &errJSON) == nil {
				if errJSON["code"] == "chat_choice_shown" {
					return "⚠️ 检测到对话分支未选择，请重试或新建会话。", nil
				}
			}
			continue
		}

		payload := line[6:]
		if strings.TrimSpace(payload) == "" {
			continue
		}

		var j map[string]any
		if json.Unmarshal([]byte(payload), &j) != nil {
			continue
		}

		// 记录消息 ID
		if v, ok := j["msg_id"].(string); ok {
			currentMsgID = v
		}

		// 分支内容 c: []
		if cList, ok := j["c"].([]any); ok {
			for _, item := range cList {
				mObj, _ := item.(map[string]any)
				idx, _ := mObj["c"].(float64)

				if int(idx) == 0 { // 选第一个
					if v, ok := mObj["v"].(string); ok {
						result.WriteString(v)
					}
				}
			}
			continue
		}

		// 普通内容 v
		if v, ok := j["v"].(string); ok {
			result.WriteString(v)
		}
	}

	if currentMsgID != "" {
		c.SendChoice(currentMsgID)
	}

	return result.String(), nil
}

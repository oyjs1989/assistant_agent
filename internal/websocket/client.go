package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"assistant_agent/internal/logger"

	"github.com/gorilla/websocket"
)

// Message 消息结构
type Message struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	ID      string      `json:"id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Client WebSocket 客户端
type Client struct {
	url       string
	token     string
	conn      *websocket.Conn
	connected bool
	mu        sync.RWMutex
}

// NewClient 创建新的 WebSocket 客户端
func NewClient(url, token string) (*Client, error) {
	return &Client{
		url:   url,
		token: token,
	}, nil
}

// Connect 连接到服务器
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// 创建请求头
	headers := http.Header{}
	if c.token != "" {
		headers.Add("Authorization", "Bearer "+c.token)
	}

	// 建立连接
	conn, _, err := websocket.DefaultDialer.Dial(c.url, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	c.conn = conn
	c.connected = true

	logger.Info("Connected to server via WebSocket")
	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false

	logger.Info("Disconnected from server")
}

// Stop 停止客户端
func (c *Client) Stop() {
	c.Disconnect()
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.conn != nil
}

// GetURL 获取服务器 URL
func (c *Client) GetURL() string {
	return c.url
}

// Send 发送消息（别名方法）
func (c *Client) Send(msgType string, data interface{}) error {
	return c.SendMessage(msgType, data)
}

// SendMessage 发送消息
func (c *Client) SendMessage(msgType string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	msg := Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	// 序列化消息
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// 发送消息
	if err := c.conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		c.connected = false
		return fmt.Errorf("failed to send message: %v", err)
	}

	logger.Debugf("Sent message: %s", msgType)
	return nil
}

// SendHeartbeat 发送心跳
func (c *Client) SendHeartbeat(status interface{}) error {
	return c.SendMessage("heartbeat", status)
}

// SendSystemInfo 发送系统信息
func (c *Client) SendSystemInfo(info interface{}) error {
	return c.SendMessage("system_info", info)
}

// SendCommandResult 发送命令执行结果
func (c *Client) SendCommandResult(result interface{}) error {
	return c.SendMessage("command_result", result)
}

// SendTaskResult 发送任务执行结果
func (c *Client) SendTaskResult(result interface{}) error {
	return c.SendMessage("task_result", result)
}

// HandleMessages 处理接收到的消息
func (c *Client) HandleMessages(handler func(string, interface{}) error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	for {
		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("WebSocket connection closed unexpectedly: %v", err)
			}
			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()
			return
		}

		// 解析消息
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Errorf("Failed to unmarshal message: %v", err)
			continue
		}

		logger.Debugf("Received message: %s", msg.Type)

		// 处理消息
		if err := handler(msg.Type, msg.Data); err != nil {
			logger.Errorf("Failed to handle message %s: %v", msg.Type, err)
		}
	}
}

// SendPing 发送 ping
func (c *Client) SendPing() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	return c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
}

// SetPongHandler 设置 pong 处理器
func (c *Client) SetPongHandler(handler func(string) error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn != nil {
		c.conn.SetPongHandler(handler)
	}
}

// SetCloseHandler 设置关闭处理器
func (c *Client) SetCloseHandler(handler func(int, string) error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn != nil {
		c.conn.SetCloseHandler(handler)
	}
}

// Receive 接收消息
func (c *Client) Receive() (string, interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return "", nil, fmt.Errorf("not connected")
	}

	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return "", nil, err
	}

	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		return "", nil, err
	}

	return msg.Type, msg.Data, nil
} 
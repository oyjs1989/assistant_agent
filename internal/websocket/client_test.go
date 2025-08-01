package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"assistant_agent/internal/config"
	"assistant_agent/internal/logger"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// 初始化配置和日志
	config.Init()
	logger.Init()
}

func TestNewClient(t *testing.T) {
	// 测试创建 WebSocket 客户端
	url := "ws://localhost:8080/ws"
	token := "test-token"
	client, err := NewClient(url, token)
	require.NoError(t, err)

	assert.NotNil(t, client)
	assert.Equal(t, url, client.url)
	assert.Equal(t, token, client.token)
	assert.False(t, client.connected)
}

func TestClientConnect(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 升级到 WebSocket
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// 简单的消息处理
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// 回显消息
			conn.WriteMessage(websocket.TextMessage, message)
		}
	}))
	defer server.Close()

	// 创建 WebSocket URL
	wsURL := "ws" + server.URL[4:] + "/ws"

	// 创建客户端
	client, err := NewClient(wsURL, "test-token")
	require.NoError(t, err)

	// 连接
	err = client.Connect()
	require.NoError(t, err)
	assert.True(t, client.IsConnected())

	// 断开连接
	client.Disconnect()
	assert.False(t, client.IsConnected())
}

func TestClientConnectInvalidURL(t *testing.T) {
	// 测试连接无效 URL
	client, err := NewClient("invalid-url", "test-token")
	require.NoError(t, err)

	err = client.Connect()
	assert.Error(t, err)
	assert.False(t, client.IsConnected())
}

func TestClientSendMessage(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// 读取消息并验证
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// 解析消息
		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			return
		}

		// 验证消息类型
		assert.Equal(t, "test", msg.Type)
		assert.Equal(t, "test data", msg.Data)
	}))
	defer server.Close()

	// 创建 WebSocket URL
	wsURL := "ws" + server.URL[4:] + "/ws"

	// 创建客户端
	client, err := NewClient(wsURL, "test-token")
	require.NoError(t, err)

	// 连接
	err = client.Connect()
	require.NoError(t, err)
	defer client.Disconnect()

	// 发送消息
	err = client.SendMessage("test", "test data")
	require.NoError(t, err)

	// 等待消息处理
	time.Sleep(100 * time.Millisecond)
}

func TestClientSendMessageNotConnected(t *testing.T) {
	// 创建客户端但不连接
	client, err := NewClient("ws://localhost:8080/ws", "test-token")
	require.NoError(t, err)

	// 尝试发送消息
	err = client.SendMessage("test", "test data")
	assert.Error(t, err)
}

func TestClientGetURL(t *testing.T) {
	// 测试获取 URL
	url := "ws://localhost:8080/ws"
	client, err := NewClient(url, "test-token")
	require.NoError(t, err)

	assert.Equal(t, url, client.GetURL())
}

func TestMessageStructure(t *testing.T) {
	// 测试消息结构
	msg := Message{
		Type:      "test",
		Data:      "test data",
		ID:        "test-id",
		Timestamp: time.Now(),
	}

	assert.Equal(t, "test", msg.Type)
	assert.Equal(t, "test data", msg.Data)
	assert.Equal(t, "test-id", msg.ID)
	assert.NotZero(t, msg.Timestamp)
}

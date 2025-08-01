# Assistant Agent

一个轻量级的云原生代理系统，支持任务执行、插件扩展和 WebSocket 通信。

## 功能特性

- 🔌 **插件化架构**：支持动态加载和热插拔插件
- 🚀 **任务执行**：支持 Shell、PowerShell、Container 等多种执行环境
- 🌐 **WebSocket 通信**：实时双向通信
- 💓 **心跳检测**：健康状态监控
- 🔒 **安全认证**：Token 认证和 SSL 支持
- 📊 **状态管理**：完整的任务和系统状态跟踪
- 🔧 **配置管理**：灵活的配置系统

## 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/your-org/assistant_agent.git
cd assistant_agent

# 安装依赖
go mod download

# 编译
go build -o assistant_agent main.go
```

### 配置

创建配置文件 `config.yaml`：

```yaml
# 服务器配置
server:
  host: "localhost"
  port: 8080
  url: "ws://localhost:8080/ws"

# Agent 配置
agent:
  id: "" # 留空将自动生成
  name: "assistant-agent"
  version: "1.0.0"
  heartbeat: 30 # 心跳间隔（秒）
  max_retries: 3
  retry_delay: 5 # 重试延迟（秒）

# 日志配置
logging:
  level: "info" # debug, info, warn, error
  format: "json" # json, text

# 安全配置
security:
  token: "" # 认证令牌
  cert_file: "" # SSL 证书文件路径
  key_file: "" # SSL 密钥文件路径
  verify_ssl: true
```

### 运行

```bash
# 启动服务
./assistant_agent

# 或者指定配置文件
./assistant_agent -config config.yaml
```

## 项目结构

```
assistant_agent/
├── cmd/                    # 命令行工具
├── internal/               # 内部包
│   ├── agent/             # 主代理逻辑
│   ├── config/            # 配置管理
│   ├── executor/          # 任务执行器
│   ├── heartbeat/         # 心跳检测
│   ├── logger/            # 日志系统
│   ├── plugin/            # 插件系统
│   ├── state/             # 状态管理
│   ├── sysinfo/           # 系统信息收集
│   └── websocket/         # WebSocket通信
├── pkg/                   # 公共包
├── docs/                  # 文档
├── examples/              # 示例代码
├── tests/                 # 测试文件
├── config.yaml           # 配置文件
├── go.mod                # Go模块文件
└── README.md             # 项目说明
```

## 插件开发

### 插件接口

```go
type Plugin interface {
    Name() string
    Version() string
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
    Validate(params map[string]interface{}) error
    GetCapabilities() []string
}
```

### 示例插件

```go
package main

import (
    "context"
    "assistant_agent/internal/plugin"
)

type HelloPlugin struct{}

func (p *HelloPlugin) Name() string {
    return "hello"
}

func (p *HelloPlugin) Version() string {
    return "1.0.0"
}

func (p *HelloPlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    name := params["name"].(string)
    return map[string]string{"message": "Hello, " + name + "!"}, nil
}

func (p *HelloPlugin) Validate(params map[string]interface{}) error {
    if _, ok := params["name"]; !ok {
        return errors.New("name parameter is required")
    }
    return nil
}

func (p *HelloPlugin) GetCapabilities() []string {
    return []string{"greeting"}
}
```

## API 文档

### WebSocket API

#### 连接

```javascript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onopen = function () {
  console.log("Connected to Assistant Agent");
};

ws.onmessage = function (event) {
  const message = JSON.parse(event.data);
  console.log("Received:", message);
};
```

#### 发送命令

```javascript
// 执行Shell命令
ws.send(
  JSON.stringify({
    type: "command",
    data: {
      type: "shell",
      script: 'echo "Hello World"',
      timeout: 30,
    },
  })
);

// 执行PowerShell命令
ws.send(
  JSON.stringify({
    type: "command",
    data: {
      type: "powershell",
      script: 'Write-Host "Hello World"',
      timeout: 30,
    },
  })
);
```

#### 获取系统信息

```javascript
ws.send(
  JSON.stringify({
    type: "system_info",
    data: {},
  })
);
```

## 开发指南

### 环境要求

- Go 1.21+
- Git

### 开发设置

```bash
# 克隆项目
git clone https://github.com/your-org/assistant_agent.git
cd assistant_agent

# 安装依赖
go mod download

# 运行测试
go test ./...

# 运行基准测试
go test -bench=. ./...

# 代码格式化
go fmt ./...

# 代码检查
golangci-lint run
```

### 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 优化路线图

基于对腾讯云 TAT Agent、AWS SSM Agent、阿里云 Assist Client 三款成熟产品的深入分析，我们制定了详细的优化路线图：

### 📋 [优化路线图](OPTIMIZATION_ROADMAP.md)

详细的优化方向和判断标准，包括架构重构、功能增强、性能优化等。

### 🛠️ [技术实现指南](IMPLEMENTATION_GUIDE.md)

具体的技术实现指导，包含代码示例、接口设计和实现步骤。

### 📅 [实施计划](IMPLEMENTATION_PLAN.md)

详细的实施计划，包含里程碑、时间安排和成功标准。

### 优化重点

1. **架构层面**：插件化架构重构、事件驱动机制
2. **执行引擎**：任务生命周期管理、多执行环境支持
3. **通信机制**：多协议支持、消息队列集成
4. **安全机制**：认证授权体系、加密通信
5. **监控运维**：指标收集、日志系统、容器化支持

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 支持

如果您有任何问题或建议，请：

- 提交 [Issue](https://github.com/your-org/assistant_agent/issues)
- 发送邮件到 support@your-org.com
- 查看 [文档](docs/)

## 致谢

感谢以下开源项目的支持：

- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [Viper](https://github.com/spf13/viper)
- [Logrus](https://github.com/sirupsen/logrus)
- [Cron](https://github.com/robfig/cron)

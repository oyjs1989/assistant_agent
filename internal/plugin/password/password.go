package password

import (
	"crypto/aes"
	"crypto/cipher"
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"assistant_agent/internal/plugin"

	"golang.org/x/crypto/pbkdf2"
)

// PasswordPlugin 密码管理插件
type PasswordPlugin struct {
	ctx       *plugin.PluginContext
	config    map[string]interface{}
	status    *plugin.PluginStatus
	passwords map[string]*PasswordEntry
	masterKey []byte
	dataFile  string
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// PasswordEntry 密码条目
type PasswordEntry struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastUsed    time.Time `json:"last_used"`
	ExpiresAt   time.Time `json:"expires_at"`
	Strength    int       `json:"strength"` // 1-10
	Notes       string    `json:"notes"`
}

// PasswordRequest 密码请求
type PasswordRequest struct {
	Title       string   `json:"title"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	ExpiresAt   string   `json:"expires_at"`
	Notes       string   `json:"notes"`
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query    string   `json:"query"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// NewPasswordPlugin 创建密码管理插件
func NewPasswordPlugin() *PasswordPlugin {
	return &PasswordPlugin{
		config:    make(map[string]interface{}),
		passwords: make(map[string]*PasswordEntry),
		stopChan:  make(chan struct{}),
		status: &plugin.PluginStatus{
			Status: "stopped",
			Metrics: map[string]interface{}{
				"total_passwords":   0,
				"weak_passwords":    0,
				"expired_passwords": 0,
			},
		},
	}
}

// Info 返回插件信息
func (p *PasswordPlugin) Info() *plugin.PluginInfo {
	return &plugin.PluginInfo{
		Name:        "password-manager",
		Version:     "1.0.0",
		Description: "Secure password management plugin",
		Author:      "Assistant Agent Team",
		License:     "MIT",
		Homepage:    "https://github.com/assistant-agent/plugins",
		Tags:        []string{"password", "security", "encryption"},
		Config: map[string]string{
			"master_password": "",
			"auto_lock":       "true",
			"lock_timeout":    "300",
			"backup_enabled":  "true",
		},
	}
}

// Init 初始化插件
func (p *PasswordPlugin) Init(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.status.Status = "initialized"

	// 设置数据文件路径
	p.dataFile = filepath.Join(ctx.Agent.GetConfig("data_dir").(string), "passwords.enc")

	// 初始化主密钥
	if err := p.initializeMasterKey(); err != nil {
		return fmt.Errorf("failed to initialize master key: %w", err)
	}

	// 加载密码数据
	if err := p.loadPasswords(); err != nil {
		p.ctx.Logger.Warnf("Failed to load passwords: %v", err)
	}

	p.ctx.Logger.Info("Password plugin initialized")
	return nil
}

// Start 启动插件
func (p *PasswordPlugin) Start() error {
	p.status.Status = "running"
	p.status.StartTime = time.Now()

	// 启动后台任务
	go p.backgroundTask()

	p.ctx.Logger.Info("Password plugin started")
	return nil
}

// Stop 停止插件
func (p *PasswordPlugin) Stop() error {
	p.status.Status = "stopped"
	close(p.stopChan)

	// 保存密码数据
	if err := p.savePasswords(); err != nil {
		p.ctx.Logger.Errorf("Failed to save passwords: %v", err)
	}

	p.ctx.Logger.Info("Password plugin stopped")
	return nil
}

// HandleCommand 处理命令
func (p *PasswordPlugin) HandleCommand(command string, args map[string]interface{}) (interface{}, error) {
	switch command {
	case "add":
		return p.handleAdd(args)
	case "get":
		return p.handleGet(args)
	case "update":
		return p.handleUpdate(args)
	case "delete":
		return p.handleDelete(args)
	case "list":
		return p.handleList(args)
	case "search":
		return p.handleSearch(args)
	case "generate":
		return p.handleGenerate(args)
	case "check_strength":
		return p.handleCheckStrength(args)
	case "export":
		return p.handleExport(args)
	case "import":
		return p.handleImport(args)
	default:
		return nil, plugin.ErrInvalidCommand
	}
}

// HandleEvent 处理事件
func (p *PasswordPlugin) HandleEvent(eventType string, data map[string]interface{}) error {
	switch eventType {
	case "password_expired":
		return p.handlePasswordExpired(data)
	case "weak_password_detected":
		return p.handleWeakPasswordDetected(data)
	case "security_alert":
		return p.handleSecurityAlert(data)
	default:
		return plugin.ErrInvalidEvent
	}
}

// Status 返回插件状态
func (p *PasswordPlugin) Status() *plugin.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.status.Metrics["total_passwords"] = len(p.passwords)

	weakCount := 0
	expiredCount := 0
	now := time.Now()

	for _, entry := range p.passwords {
		if entry.Strength < 5 {
			weakCount++
		}
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			expiredCount++
		}
	}

	p.status.Metrics["weak_passwords"] = weakCount
	p.status.Metrics["expired_passwords"] = expiredCount

	return p.status
}

// Health 健康检查
func (p *PasswordPlugin) Health() error {
	if p.status.Status != "running" {
		return fmt.Errorf("plugin not running")
	}
	return nil
}

// GetConfig 获取配置
func (p *PasswordPlugin) GetConfig() map[string]interface{} {
	return p.config
}

// SetConfig 设置配置
func (p *PasswordPlugin) SetConfig(config map[string]interface{}) error {
	p.config = config
	return nil
}

// handleAdd 处理添加密码命令
func (p *PasswordPlugin) handleAdd(args map[string]interface{}) (interface{}, error) {
	title, ok := args["title"].(string)
	if !ok {
		return nil, fmt.Errorf("title is required")
	}

	username, _ := args["username"].(string)
	password, _ := args["password"].(string)
	url, _ := args["url"].(string)
	description, _ := args["description"].(string)
	category, _ := args["category"].(string)

	// 生成密码ID
	id := p.generateID()

	// 创建密码条目
	entry := &PasswordEntry{
		ID:          id,
		Title:       title,
		Username:    username,
		Password:    password,
		URL:         url,
		Description: description,
		Category:    category,
		Tags:        p.parseTags(args["tags"]),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Strength:    p.calculatePasswordStrength(password),
		Notes:       args["notes"].(string),
	}

	// 设置过期时间
	if expiresAt, ok := args["expires_at"].(string); ok && expiresAt != "" {
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			entry.ExpiresAt = t
		}
	}

	// 添加到密码库
	p.mu.Lock()
	p.passwords[id] = entry
	p.mu.Unlock()

	// 保存到文件
	if err := p.savePasswords(); err != nil {
		p.ctx.Logger.Errorf("Failed to save password: %v", err)
	}

	p.ctx.Logger.Infof("Password added: %s", title)

	return map[string]interface{}{
		"id":      id,
		"title":   title,
		"message": "Password added successfully",
	}, nil
}

// handleGet 处理获取密码命令
func (p *PasswordPlugin) handleGet(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.RLock()
	entry, exists := p.passwords[id]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("password not found")
	}

	// 更新最后使用时间
	entry.LastUsed = time.Now()

	return entry, nil
}

// handleUpdate 处理更新密码命令
func (p *PasswordPlugin) handleUpdate(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	entry, exists := p.passwords[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("password not found")
	}

	// 更新字段
	if title, ok := args["title"].(string); ok {
		entry.Title = title
	}
	if username, ok := args["username"].(string); ok {
		entry.Username = username
	}
	if password, ok := args["password"].(string); ok {
		entry.Password = password
		entry.Strength = p.calculatePasswordStrength(password)
	}
	if url, ok := args["url"].(string); ok {
		entry.URL = url
	}
	if description, ok := args["description"].(string); ok {
		entry.Description = description
	}
	if category, ok := args["category"].(string); ok {
		entry.Category = category
	}
	if notes, ok := args["notes"].(string); ok {
		entry.Notes = notes
	}

	entry.UpdatedAt = time.Now()
	p.mu.Unlock()

	// 保存到文件
	if err := p.savePasswords(); err != nil {
		p.ctx.Logger.Errorf("Failed to save password: %v", err)
	}

	p.ctx.Logger.Infof("Password updated: %s", entry.Title)

	return map[string]interface{}{
		"id":      id,
		"message": "Password updated successfully",
	}, nil
}

// handleDelete 处理删除密码命令
func (p *PasswordPlugin) handleDelete(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	entry, exists := p.passwords[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("password not found")
	}

	delete(p.passwords, id)
	p.mu.Unlock()

	// 保存到文件
	if err := p.savePasswords(); err != nil {
		p.ctx.Logger.Errorf("Failed to save passwords: %v", err)
	}

	p.ctx.Logger.Infof("Password deleted: %s", entry.Title)

	return map[string]interface{}{
		"id":      id,
		"message": "Password deleted successfully",
	}, nil
}

// handleList 处理列表命令
func (p *PasswordPlugin) handleList(args map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entries := make([]*PasswordEntry, 0, len(p.passwords))
	for _, entry := range p.passwords {
		// 不返回实际密码
		safeEntry := *entry
		safeEntry.Password = "***"
		entries = append(entries, &safeEntry)
	}

	return map[string]interface{}{
		"passwords": entries,
		"count":     len(entries),
	}, nil
}

// handleSearch 处理搜索命令
func (p *PasswordPlugin) handleSearch(args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	category, _ := args["category"].(string)
	tags := p.parseTags(args["tags"])

	p.mu.RLock()
	defer p.mu.RUnlock()

	var results []*PasswordEntry
	for _, entry := range p.passwords {
		// 检查查询条件
		if query != "" {
			if !p.matchesQuery(entry, query) {
				continue
			}
		}

		if category != "" && entry.Category != category {
			continue
		}

		if len(tags) > 0 {
			if !p.matchesTags(entry, tags) {
				continue
			}
		}

		// 不返回实际密码
		safeEntry := *entry
		safeEntry.Password = "***"
		results = append(results, &safeEntry)
	}

	return map[string]interface{}{
		"results": results,
		"count":   len(results),
	}, nil
}

// handleGenerate 处理生成密码命令
func (p *PasswordPlugin) handleGenerate(args map[string]interface{}) (interface{}, error) {
	length, ok := args["length"].(float64)
	if !ok {
		length = 16
	}

	includeUppercase, _ := args["include_uppercase"].(bool)
	includeLowercase, _ := args["include_lowercase"].(bool)
	includeNumbers, _ := args["include_numbers"].(bool)
	includeSymbols, _ := args["include_symbols"].(bool)

	password := p.generatePassword(int(length), includeUppercase, includeLowercase, includeNumbers, includeSymbols)
	strength := p.calculatePasswordStrength(password)

	return map[string]interface{}{
		"password": password,
		"strength": strength,
		"length":   len(password),
	}, nil
}

// handleCheckStrength 处理检查密码强度命令
func (p *PasswordPlugin) handleCheckStrength(args map[string]interface{}) (interface{}, error) {
	password, ok := args["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password is required")
	}

	strength := p.calculatePasswordStrength(password)
	feedback := p.getPasswordFeedback(password)

	return map[string]interface{}{
		"strength": strength,
		"feedback": feedback,
	}, nil
}

// handleExport 处理导出命令
func (p *PasswordPlugin) handleExport(args map[string]interface{}) (interface{}, error) {
	format, _ := args["format"].(string)
	if format == "" {
		format = "json"
	}

	p.mu.RLock()
	entries := make([]*PasswordEntry, 0, len(p.passwords))
	for _, entry := range p.passwords {
		entries = append(entries, entry)
	}
	p.mu.RUnlock()

	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(entries, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, err
	}

	// 加密导出数据
	encryptedData, err := p.encrypt(data)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"data":   base64.StdEncoding.EncodeToString(encryptedData),
		"format": format,
		"count":  len(entries),
	}, nil
}

// handleImport 处理导入命令
func (p *PasswordPlugin) handleImport(args map[string]interface{}) (interface{}, error) {
	data, ok := args["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data is required")
	}

	format, _ := args["format"].(string)
	if format == "" {
		format = "json"
	}

	// 解密数据
	encryptedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	decryptedData, err := p.decrypt(encryptedData)
	if err != nil {
		return nil, err
	}

	var entries []*PasswordEntry
	switch format {
	case "json":
		err = json.Unmarshal(decryptedData, &entries)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, err
	}

	// 导入密码
	imported := 0
	for _, entry := range entries {
		if entry.ID == "" {
			entry.ID = p.generateID()
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = time.Now()
		}
		entry.UpdatedAt = time.Now()

		p.mu.Lock()
		p.passwords[entry.ID] = entry
		p.mu.Unlock()
		imported++
	}

	// 保存到文件
	if err := p.savePasswords(); err != nil {
		p.ctx.Logger.Errorf("Failed to save imported passwords: %v", err)
	}

	return map[string]interface{}{
		"imported": imported,
		"message":  "Import completed successfully",
	}, nil
}

// 辅助方法

// initializeMasterKey 初始化主密钥
func (p *PasswordPlugin) initializeMasterKey() error {
	// 从配置或环境变量获取主密码
	masterPassword := p.config["master_password"].(string)
	if masterPassword == "" {
		masterPassword = os.Getenv("PASSWORD_MASTER_KEY")
	}

	if masterPassword == "" {
		// 生成随机主密钥
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return err
		}
		p.masterKey = key
	} else {
		// 从密码派生密钥
		salt := []byte("assistant_agent_salt")
		p.masterKey = pbkdf2.Key([]byte(masterPassword), salt, 10000, 32, sha256.New)
	}

	return nil
}

// loadPasswords 加载密码数据
func (p *PasswordPlugin) loadPasswords() error {
	if !p.ctx.Agent.FileExists(p.dataFile) {
		return nil
	}

	data, err := p.ctx.Agent.ReadFile(p.dataFile)
	if err != nil {
		return err
	}

	// 解密数据
	decryptedData, err := p.decrypt(data)
	if err != nil {
		return err
	}

	var entries []*PasswordEntry
	if err := json.Unmarshal(decryptedData, &entries); err != nil {
		return err
	}

	p.mu.Lock()
	for _, entry := range entries {
		p.passwords[entry.ID] = entry
	}
	p.mu.Unlock()

	return nil
}

// savePasswords 保存密码数据
func (p *PasswordPlugin) savePasswords() error {
	p.mu.RLock()
	entries := make([]*PasswordEntry, 0, len(p.passwords))
	for _, entry := range p.passwords {
		entries = append(entries, entry)
	}
	p.mu.RUnlock()

	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	// 加密数据
	encryptedData, err := p.encrypt(data)
	if err != nil {
		return err
	}

	return p.ctx.Agent.WriteFile(p.dataFile, encryptedData)
}

// encrypt 加密数据
func (p *PasswordPlugin) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crypto_rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

// decrypt 解密数据
func (p *PasswordPlugin) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// generateID 生成唯一ID
func (p *PasswordPlugin) generateID() string {
	b := make([]byte, 16)
	crypto_rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// generatePassword 生成密码
func (p *PasswordPlugin) generatePassword(length int, uppercase, lowercase, numbers, symbols bool) string {
	const (
		upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowerChars  = "abcdefghijklmnopqrstuvwxyz"
		numberChars = "0123456789"
		symbolChars = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	var chars string
	if uppercase {
		chars += upperChars
	}
	if lowercase {
		chars += lowerChars
	}
	if numbers {
		chars += numberChars
	}
	if symbols {
		chars += symbolChars
	}

	if chars == "" {
		chars = lowerChars + numberChars
	}

	password := make([]byte, length)
	for i := range password {
		password[i] = chars[rand.Intn(len(chars))]
	}

	return string(password)
}

// calculatePasswordStrength 计算密码强度
func (p *PasswordPlugin) calculatePasswordStrength(password string) int {
	if len(password) == 0 {
		return 0
	}

	score := 0

	// 长度分数
	if len(password) >= 8 {
		score += 2
	}
	if len(password) >= 12 {
		score += 2
	}
	if len(password) >= 16 {
		score += 1
	}

	// 字符类型分数
	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSymbol := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		default:
			hasSymbol = true
		}
	}

	if hasUpper {
		score += 1
	}
	if hasLower {
		score += 1
	}
	if hasNumber {
		score += 1
	}
	if hasSymbol {
		score += 2
	}

	// 限制分数范围
	if score > 10 {
		score = 10
	}

	return score
}

// getPasswordFeedback 获取密码反馈
func (p *PasswordPlugin) getPasswordFeedback(password string) []string {
	var feedback []string

	if len(password) < 8 {
		feedback = append(feedback, "Password is too short")
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSymbol := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		default:
			hasSymbol = true
		}
	}

	if !hasUpper {
		feedback = append(feedback, "Add uppercase letters")
	}
	if !hasLower {
		feedback = append(feedback, "Add lowercase letters")
	}
	if !hasNumber {
		feedback = append(feedback, "Add numbers")
	}
	if !hasSymbol {
		feedback = append(feedback, "Add symbols")
	}

	return feedback
}

// parseTags 解析标签
func (p *PasswordPlugin) parseTags(tags interface{}) []string {
	if tags == nil {
		return nil
	}

	switch v := tags.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, len(v))
		for i, tag := range v {
			if str, ok := tag.(string); ok {
				result[i] = str
			}
		}
		return result
	default:
		return nil
	}
}

// matchesQuery 检查是否匹配查询
func (p *PasswordPlugin) matchesQuery(entry *PasswordEntry, query string) bool {
	query = strings.ToLower(query)

	return strings.Contains(strings.ToLower(entry.Title), query) ||
		strings.Contains(strings.ToLower(entry.Username), query) ||
		strings.Contains(strings.ToLower(entry.URL), query) ||
		strings.Contains(strings.ToLower(entry.Description), query)
}

// matchesTags 检查是否匹配标签
func (p *PasswordPlugin) matchesTags(entry *PasswordEntry, tags []string) bool {
	for _, tag := range tags {
		found := false
		for _, entryTag := range entry.Tags {
			if entryTag == tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// backgroundTask 后台任务
func (p *PasswordPlugin) backgroundTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查过期密码
			p.checkExpiredPasswords()
		case <-p.stopChan:
			return
		}
	}
}

// checkExpiredPasswords 检查过期密码
func (p *PasswordPlugin) checkExpiredPasswords() {
	p.mu.RLock()
	var expired []*PasswordEntry
	now := time.Now()

	for _, entry := range p.passwords {
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			expired = append(expired, entry)
		}
	}
	p.mu.RUnlock()

	for _, entry := range expired {
		p.ctx.Logger.Warnf("Password expired: %s", entry.Title)
		// 发送过期事件
		p.ctx.Agent.NotifyEvent("password_expired", map[string]interface{}{
			"id":    entry.ID,
			"title": entry.Title,
		})
	}
}

// 事件处理方法
func (p *PasswordPlugin) handlePasswordExpired(data map[string]interface{}) error {
	p.ctx.Logger.Info("Password expired event received")
	return nil
}

func (p *PasswordPlugin) handleWeakPasswordDetected(data map[string]interface{}) error {
	p.ctx.Logger.Info("Weak password detected event received")
	return nil
}

func (p *PasswordPlugin) handleSecurityAlert(data map[string]interface{}) error {
	p.ctx.Logger.Info("Security alert event received")
	return nil
}

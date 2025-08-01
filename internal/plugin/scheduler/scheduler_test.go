package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSchedulerPlugin(t *testing.T) {
	plugin := NewSchedulerPlugin()
	assert.NotNil(t, plugin)
	assert.NotNil(t, plugin.scheduler)
	assert.NotNil(t, plugin.tasks)
	assert.NotNil(t, plugin.stopChan)
}

func TestSchedulerPluginInfo(t *testing.T) {
	plugin := NewSchedulerPlugin()
	info := plugin.Info()

	assert.Equal(t, "task-scheduler", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "Cron-based task scheduler plugin", info.Description)
	assert.Contains(t, info.Tags, "scheduler")
	assert.Contains(t, info.Tags, "cron")
	assert.Contains(t, info.Tags, "task")
}

func TestSchedulerPluginFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, "scheduler", factory.GetPluginType())

	plugin, err := factory.CreatePlugin(nil)
	assert.NoError(t, err)
	assert.NotNil(t, plugin)

	// 验证插件类型
	info := plugin.Info()
	assert.Equal(t, "task-scheduler", info.Name)
}

func TestSchedulerPluginBasicCommands(t *testing.T) {
	plugin := NewSchedulerPlugin()

	// 测试列出任务（应该返回空列表）
	result, err := plugin.HandleCommand("list_tasks", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 测试获取下次运行时间（应该返回空映射）
	result, err = plugin.HandleCommand("get_next_runs", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 测试无效命令
	_, err = plugin.HandleCommand("invalid_command", nil)
	assert.Error(t, err)
}

func TestSchedulerPluginTaskManagement(t *testing.T) {
	plugin := NewSchedulerPlugin()

	// 测试添加任务（不启用，避免调度器依赖）
	result, err := plugin.HandleCommand("add_task", map[string]interface{}{
		"name":      "test-task",
		"cron_expr": "*/1 * * * *",
		"command":   "echo 'hello'",
		"type":      "shell",
		"enabled":   false, // 明确设置为禁用
	})

	// 添加调试信息
	if err != nil {
		t.Logf("Add task error: %v", err)
	} else {
		t.Logf("Add task result: %+v", result)
	}

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 验证任务已添加
	result, err = plugin.HandleCommand("list_tasks", nil)
	if err != nil {
		t.Logf("List tasks error: %v", err)
	} else {
		t.Logf("List tasks result: %+v", result)
	}

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 处理返回的 map 结构
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok, "Expected map[string]interface{}")

	tasks, ok := resultMap["tasks"].([]*TaskInfo)
	t.Logf("Tasks type assertion: %v, tasks: %+v", ok, tasks)

	assert.True(t, ok)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "test-task", tasks[0].Name)
	assert.False(t, tasks[0].Enabled)

	// 获取任务ID
	taskID := tasks[0].ID

	// 测试获取任务详情
	result, err = plugin.HandleCommand("get_task", map[string]interface{}{
		"id": taskID,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	task, ok := result.(*TaskInfo)
	assert.True(t, ok)
	assert.Equal(t, "test-task", task.Name)
	assert.Equal(t, taskID, task.ID)

	// 测试删除任务
	result, err = plugin.HandleCommand("remove_task", map[string]interface{}{
		"id": taskID,
	})
	assert.NoError(t, err)

	// 验证任务已删除
	result, err = plugin.HandleCommand("list_tasks", nil)
	assert.NoError(t, err)
	resultMap, ok = result.(map[string]interface{})
	assert.True(t, ok)

	tasks, ok = resultMap["tasks"].([]*TaskInfo)
	assert.True(t, ok)
	assert.Len(t, tasks, 0)
}

func TestSchedulerPluginGenerateID(t *testing.T) {
	plugin := NewSchedulerPlugin()
	
	// 测试ID生成
	id1 := plugin.generateID()
	
	// 添加小延迟确保时间戳不同
	time.Sleep(1 * time.Millisecond)
	
	id2 := plugin.generateID()
	
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}

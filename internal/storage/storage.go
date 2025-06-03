// storage/storage.go - 配置和临时文件存储
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"codebase-syncer/pkg/logger"
)

// 文件状态常量
const (
	FILE_STATUS_ADDED    = "add"
	FILE_STATUS_MODIFIED = "modify"
	FILE_STATUS_DELETED  = "delete"
)

// codebase配置
type CodebaseConfig struct {
	ClientID     string            `json:"clientId"`
	CodebaseName string            `json:"codebaseName"`
	CodebasePath string            `json:"codebasePath"`
	CodebaseId   string            `json:"codebaseId"`
	HashTree     map[string]string `json:"hashTree"`
	LastSync     time.Time         `json:"lastSync"`
	RegisterTime time.Time         `json:"registerTime"`
}

// 同步文件信息
type SyncFile struct {
	Path   string `json:"path"`
	Hash   string `json:"hash"`
	Status string `json:"status"`
}

type ConfigManager struct {
	codebasePath    string
	codebaseConfigs map[string]*CodebaseConfig // 存储所有codebase 配置
	logger          logger.Logger
	mutex           sync.RWMutex
}

// NewStorageManager 创建一个新的配置管理器
func NewStorageManager(cacheDir string, logger logger.Logger) *ConfigManager {
	// 确保codebase目录存在
	codebasePath := filepath.Join(cacheDir, "codebase")
	if _, err := os.Stat(codebasePath); os.IsNotExist(err) {
		if err := os.MkdirAll(codebasePath, 0755); err != nil {
			logger.Fatal("无法创建codebase目录: %v", err)
		}
	}

	// 初始化 codebaseConfigs map
	sm := &ConfigManager{
		codebasePath:    codebasePath,
		logger:          logger,
		codebaseConfigs: make(map[string]*CodebaseConfig),
	}

	sm.loadAllConfigs()
	return sm
}

// GetConfigs 获取所有项目配置
func (cm *ConfigManager) GetCodebaseConfigs() map[string]*CodebaseConfig {
	return cm.codebaseConfigs
}

// GetCodebaseConfig 加载codebase 配置
// 优先从内存配置中查找，找不到再从文件系统加载
func (cm *ConfigManager) GetCodebaseConfig(codebaseId string) (*CodebaseConfig, error) {
	cm.mutex.RLock()
	config, exists := cm.codebaseConfigs[codebaseId]
	cm.mutex.RUnlock()

	if exists {
		return config, nil
	}

	// 内存中没有查到，尝试从文件加载
	config, err := cm.loadCodebaseConfig(codebaseId)
	if err != nil {
		return nil, err
	}

	cm.mutex.Lock()
	cm.codebaseConfigs[codebaseId] = config
	cm.mutex.Unlock()

	return config, nil
}

// 加载所有codebase 配置文件
func (cm *ConfigManager) loadAllConfigs() {
	files, err := os.ReadDir(cm.codebasePath)
	if err != nil {
		if !os.IsNotExist(err) {
			cm.logger.Error("读取codebase目录失败: %v", err)
		}
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		config, err := cm.loadCodebaseConfig(file.Name())
		if err != nil {
			cm.logger.Error("加载codebase文件 %s 失败: %v", file.Name(), err)
			continue
		}
		cm.codebaseConfigs[file.Name()] = config
	}
}

// loadCodebaseConfig 加载codebase 配置文件
func (cm *ConfigManager) loadCodebaseConfig(codebaseId string) (*CodebaseConfig, error) {
	cm.logger.Info("加载codebase文件内容: %s", codebaseId)

	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	filePath := filepath.Join(cm.codebasePath, codebaseId)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("codebase文件不存在: %s", filePath)
		}
		return nil, fmt.Errorf("读取codebase文件失败: %v", err)
	}

	var config CodebaseConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析codebase文件失败: %v", err)
	}

	if config.CodebaseId != codebaseId {
		return nil, fmt.Errorf("codebase目录中的coebase文件ID不匹配: 期望 %s，实际 %s",
			codebaseId, config.CodebaseId)
	}

	cm.logger.Info("成功加载codebase文件，上次同步时间: %s",
		config.LastSync.Format(time.RFC3339))

	return &config, nil
}

// SaveCodebaseConfig 保存项目配置
func (cm *ConfigManager) SaveCodebaseConfig(config *CodebaseConfig) error {
	cm.logger.Info("保存项目配置: %s", config.CodebasePath)

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	filePath := filepath.Join(cm.codebasePath, config.CodebaseId)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	// 原子性更新内存配置
	cm.codebaseConfigs[config.CodebaseId] = config
	cm.logger.Info("项目配置保存成功, path: %s, codebaseId: %s", filePath, config.CodebaseId)
	return nil
}

// DeleteCodebaseConfig 删除codebase 配置
func (cm *ConfigManager) DeleteCodebaseConfig(codebaseId string) error {
	cm.logger.Info("删除codebase配置: %s", codebaseId)

	filePath := filepath.Join(cm.codebasePath, codebaseId)

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	exists := cm.codebaseConfigs[codebaseId] != nil

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if exists {
			delete(cm.codebaseConfigs, codebaseId)
			cm.logger.Info("仅内存中的codebase配置已删除: %s", codebaseId)
		}
		return nil
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除配置文件失败: %v", err)
	}

	// 文件删除成功后才删除内存中的配置
	if exists {
		delete(cm.codebaseConfigs, codebaseId)
		cm.logger.Info("codebase配置已删除: %s (文件+内存)", filePath)
	} else {
		cm.logger.Info("codebase文件已删除: %s (仅文件)", filePath)
	}
	return nil
}

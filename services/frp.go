package services

import (
	"os"

	frpconfig "github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/v1/validation"

	"github.com/koho/frpmgr/pkg/util"
)

func deleteFrpFiles(serviceName, configPath, logFile string) {
	// Delete logs
	if logs, _, err := util.FindLogFiles(logFile); err == nil {
		util.DeleteFiles(logs)
	}
	// Delete config file
	os.Remove(configPath)
	// Delete service
	m, err := serviceManager()
	if err != nil {
		return
	}
	defer m.Disconnect()
	service, err := m.OpenService(serviceName)
	if err != nil {
		return
	}
	defer service.Close()
	service.Delete()
}

// VerifyClientConfig validates the frp client config file
func VerifyClientConfig(path string) error {
	// 在 frp v0.66.0 中，LoadClientConfig 返回 5 个值：
	// commonCfg, proxyCfgs, visitorCfgs, baseCfg, error
	cfg, proxyCfgs, visitorCfgs, _, err := frpconfig.LoadClientConfig(path, false)
	if err != nil {
		return err
	}

	// 【关键修复点】
	// frp v0.66.0 的 ValidateAllClientConfig 需要 4 个参数：
	// (common, proxies, visitors, unsafeFeatures)
	// 这里传入 nil 代表不使用特殊的“不安全特性”校验，符合 GUI 管理器的通用逻辑
	_, err = validation.ValidateAllClientConfig(cfg, proxyCfgs, visitorCfgs, nil)
	return err
}

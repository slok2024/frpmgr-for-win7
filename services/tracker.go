package services

import (
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/koho/frpmgr/pkg/consts"
)

type ConfigStateCallback func(path string, state consts.ConfigState)

type tracker struct {
	service *mgr.Service
	done    sync.WaitGroup
	once    atomic.Uint32
}

var (
	trackedConfigs     = make(map[string]*tracker)
	trackedConfigsLock = sync.Mutex{}
)

// WatchConfigServices 修复签名不匹配问题
// UI 层期望返回: (func() error, error)
func WatchConfigServices(paths func() []string, cb ConfigStateCallback) (func() error, error) {
	err := trackExistingConfigs(paths, cb)
	// 返回一个返回 nil 的函数，满足 func() error 签名
	return func() error { return nil }, err
}

func trackExistingConfigs(paths func() []string, cb ConfigStateCallback) error {
	m, err := serviceManager()
	if err != nil {
		return err
	}
	for _, path := range paths() {
		trackedConfigsLock.Lock()
		if ctx := trackedConfigs[path]; ctx != nil {
			cfg, err := ctx.service.Config()
			trackedConfigsLock.Unlock()
			if (err != nil || cfg.StartType == windows.SERVICE_DISABLED) && ctx.once.CompareAndSwap(0, 1) {
				ctx.done.Done()
				cb(path, consts.ConfigStateStopped)
			}
			continue
		}
		trackedConfigsLock.Unlock()
		serviceName := ServiceNameOfClient(path)
		service, err := m.OpenService(serviceName)
		if err != nil {
			continue
		}
		go trackService(service, path, cb)
	}
	return nil
}

func trackService(service *mgr.Service, path string, cb ConfigStateCallback) {
	ctx := &tracker{service: service}
	trackedConfigsLock.Lock()
	trackedConfigs[path] = ctx
	trackedConfigsLock.Unlock()
	ctx.done.Add(1)

	updateState := func(state consts.ConfigState) {
		if state != 0 {
			cb(path, state)
		}
	}

	// ---------------------------------------------------------
	// Windows 7 兼容性核心逻辑：
	// 使用 LazyDLL 动态查找 API，避免程序启动时因找不到 DLL 入口点而弹窗
	// ---------------------------------------------------------
	modsechost := syscall.NewLazyDLL("sechost.dll")
	procSubscribe := modsechost.NewProc("SubscribeServiceChangeNotifications")
	procUnsubscribe := modsechost.NewProc("UnsubscribeServiceChangeNotifications")

	var subscription uintptr

	// 1. 检查 API 是否存在。在 Win7 下 Find() 会失败或返回 nil
	if procSubscribe.Find() != nil {
		// Win7 逻辑：因为没有订阅功能，只做一次查询，不报错，不弹窗
		status, err := service.Query()
		if err == nil {
			updateState(svcStateToConfigState(uint32(status.State)))
		}
		// 标记为已运行一次，直接退出
		ctx.once.Store(1)
		return
	}

	// 2. Win8+ 逻辑：存在该 API，正常订阅
	callback := syscall.NewCallback(func(notification uint32, context uintptr, svcHandle uintptr) uintptr {
		if notification == 4 { // SERVICE_NOTIFY_STATUS_CHANGE
			status, err := service.Query()
			if err == nil {
				updateState(svcStateToConfigState(uint32(status.State)))
			}
		}
		return 0
	})

	// 调用 SubscribeServiceChangeNotifications
	ret, _, _ := procSubscribe.Call(uintptr(service.Handle), 0, callback, 0, uintptr(unsafe.Pointer(&subscription)))

	if ret == 0 { // 成功（返回值为0表示成功，具体取决于API定义，但在Go syscall中通常如此检查）
		defer procUnsubscribe.Call(subscription)
		status, err := service.Query()
		if err == nil {
			updateState(svcStateToConfigState(uint32(status.State)))
		}
		ctx.done.Wait()
	} else {
		// 订阅失败备选逻辑
		status, err := service.Query()
		if err == nil {
			updateState(svcStateToConfigState(uint32(status.State)))
		}
	}
}

func svcStateToConfigState(s uint32) consts.ConfigState {
	switch s {
	case windows.SERVICE_STOPPED:
		return consts.ConfigStateStopped
	case windows.SERVICE_START_PENDING:
		return consts.ConfigStateStarting
	case windows.SERVICE_STOP_PENDING:
		return consts.ConfigStateStopping
	case windows.SERVICE_RUNNING:
		return consts.ConfigStateStarted
	default:
		return 0
	}
}

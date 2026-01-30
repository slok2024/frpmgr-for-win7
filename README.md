FRP Manager for win7



为了达成这个 Windows 7 完美兼容版，我们在整个过程中一共修改了 3 个核心 .go 文件（如果算上之前修复图标崩溃的 icon.go，则是 4 个）。

以下是本次“Win7 适配与修复”工程的完整修改清单复盘：

1. services/tracker.go (改动最大，最关键)
这是解决弹窗和编译报错的“主战场”。

修改前：直接调用 windows.SubscribeServiceChangeNotifications，导致 Win7 启动即弹窗“无法定位程序输入点”；且 WatchConfigServices 函数签名与 UI 层不匹配。

修改后：

引入了 LazyDLL (动态加载) 技术，手动查找 API。如果系统（Win7）没有这个 API，直接静默降级为单次查询，彻底根除弹窗。

修复了 WatchConfigServices 的返回值签名，从单返回值改为 (func() error, error)，完美适配了 ui/confpage.go 的调用逻辑。

引入了 unsafe 包以支持底层的指针操作。

2. cmd/frpmgr/main.go (最后一道防线)
为了确保万无一失，我们在程序入口处加了“保险”。

修改前：普通的程序入口。

修改后：

在 init() 函数的第一行加入了 windows.SetErrorMode。

设置了 SEM_FAILCRITICALERRORS | SEM_NOOPENFILEERRORBOX 标志位。

作用：在 Go 运行时加载任何依赖库之前，告诉操作系统“如果有 DLL 找不到，请不要弹出对话框，直接在后台处理失败”，保证了用户体验的丝滑。

3. services/frp.go (版本适配)
为了适配新版 frp v0.66.0 依赖库。

修改前：旧版的 frpconfig.LoadClientConfig 和验证逻辑参数较少。

修改后：

更新了 LoadClientConfig 的接收参数（适应 5 个返回值）。

修正了 VerifyClientConfig 函数，确保在调用 FRP 核心校验逻辑时参数对应正确。

(可选) 第 4 个文件：ui/icon.go
如果在之前的步骤中你已经修复了图标崩溃问题，那么这个文件也算在内。

作用：修复了在部分 Windows 系统上获取图标资源时的空指针引用（Nil Pointer Dereference），防止程序启动时直接闪退。



使用的编译命令：
go clean -cache
go clean -modcache
go mod tidy
go mod download
set GOARCH=386或set GOARCH=amd64
set GOOS=windows
set CGO_ENABLED=0
go generate ./...
go build -trimpath -ldflags="-s -w -H windowsgui" -o frpmgr_win7_final.exe ./cmd/frpmgr


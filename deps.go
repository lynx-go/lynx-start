package kernel

import (
	"log/slog"

	"github.com/lynx-go/lynx"
	"github.com/lynx-go/lynx-start/events"
	"github.com/lynx-go/lynx-start/uow"
	"github.com/lynx-go/lynx/eventbus"
	"github.com/uptrace/bun"
)

// Deps 是应用基础层提供给每个模块的共享依赖：数据库连接、工作单元、
// 事件发布端口、事件总线（订阅侧）、配置与日志。模块源码（安装后归应用
// 所有）在自己的 Assemble 里消费 Deps 构造仓储与用例——依赖显式、
// 代码可直读直改，不引入生成式 DI 的重装步骤。
type Deps struct {
	DB     *bun.DB
	Tx     uow.Runner
	Events events.Publisher
	// Bus 是事件总线的订阅侧：发布走 Events 端口（信封校验与编码归一），
	// 订阅跨模块事件目录（如 identity.user.password-changed）时用 Bus。
	Bus    eventbus.Bus
	Config lynx.Config
	Logger *slog.Logger
}

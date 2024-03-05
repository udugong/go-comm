# go-comm
高可用的通信服务策略(短信、邮件)

使用泛型实现通用的发送服务接口，可适配多种服务商，并提供多种策略。

只需要根据服务商的规范实现 `Sender[T any]` 接口，选择所需要的策略，初始化时依赖注入即可。

```go
type Sender[T any] interface {
	// Send 发送信息
	// biz 含糊的业务,可以是模板/模板id
	// args 参数
	// to 接收人
	Send(ctx context.Context, biz string, args T, to ...string) error
}

```



go versions
==================

`>=1.21`



# usage

下载安装：`go get github.com/udugong/go-comm`

以下方式可以考虑嵌套使用。

  * [重试](#retryable-package)
  * [限流](#ratelimit-package)
  * [故障转移](#failover-package)



# `retryable` package

该`retryable`包提供了出错重试策略。

导入`"github.com/udugong/go-comm/retryable"`

- [直接重试](#直接重试)
- [间隔重试](#间隔重试)



## 直接重试

```go
package main

import (
	"context"

	"github.com/udugong/go-comm/retryable"
)

type Args struct {
	From    string
	Subject string
	Body    string
	IsHTML  bool
}

func main() {
	// 表示创建一个最大发送3次的发送服务
	svc := retryable.NewService[Args](&testService[Args]{}, 3)
	svc.Send(context.Background(), "", Args{}, "foo@example.com")
}

// testService 模拟实现 Sender[T any] 接口
type testService[T any] struct {
	err error
}

func (svc *testService[T]) Send(_ context.Context, _ string, _ T, _ ...string) error {
	return svc.err
}

```



## 间隔重试

间隔重试允许在重试时等待一段时间。

```go
retryable.NewIntervalService[int](&testService[Args]{}, 3,
	func() time.Duration {
        // 表示重试之间随机等待 1000~1500 毫秒
        // 当然也可以固定等待一段时间 return 500 * time.Millisecond
		return time.Duration(1000+rand.Intn(501)) * time.Millisecond
	})
```



# `ratelimit` package

该`ratelimit`包提供了限流策略。但是需要实现 [Limiter](https://github.com/udugong/go-comm/blob/main/ratelimit/limiter.go#L5) 接口在初始化时注入。
在 [limiter](https://github.com/udugong/limiter) 仓库中提供了 Limiter 接口的实现。

导入`"github.com/udugong/go-comm/ratelimit"`


```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/udugong/go-comm/ratelimit"
	ratelimiter "github.com/udugong/limiter/ratelimit"
)

func main() {
	limiter := ratelimiter.NewRedisSlidingWindowLimiter(&redis.Client{}, time.Second, 1)
	sender := ratelimit.NewService[Args](&testService[Args]{}, "email", limiter)

	// 如果限流则会返回 ratelimit.ErrLimited 错误
	err := sender.Send(context.Background(), "", Args{}, "")
	if err != nil {
		if errors.Is(err, ratelimit.ErrLimited) {
			fmt.Println("限流了...")
			return
		}
		fmt.Println(err)
	}
}

type Args struct {
	From    string
	Subject string
	Body    string
	IsHTML  bool
}

// testService 模拟实现 Sender[T any] 接口
type testService[T any] struct {
	err error
}

func (svc *testService[T]) Send(_ context.Context, _ string, _ T, _ ...string) error {
	return svc.err
}

```



# `failover` package

该`failover`包提供了故障转移策略。并提供了查看与设置当前使用服务的方法。

导入`"github.com/udugong/go-comm/failover"`

- [出错故障转移](#出错故障转移)
- [连续超时故障转移](#连续超时故障转移)



## 出错故障转移

当出现 error 时会切换到下一个发送服务，直到所有的发送服务都失败则会返回 `failover.ErrAllServiceFailed` 错误，
若遇到 `context.DeadlineExceeded` 超时与 `context.Canceled` 取消则直接返回 error。 每次调用 Send 方法时会从上一个正常发送的服务开始调用。

```go
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/udugong/go-comm"
	"github.com/udugong/go-comm/failover"
)

func main() {
	sender := failover.NewService[Args]([]comm.Sender[Args]{&testService[Args]{}})

	// 如果所有服务商都失败了则会返回 failover.ErrAllServiceFailed 错误
	err := sender.Send(context.Background(), "", Args{}, "")
	if err != nil {
		switch {
		case errors.Is(err, failover.ErrAllServiceFailed):
			fmt.Println("全部服务商都失败了...")
		case errors.Is(err, context.DeadlineExceeded):
			fmt.Println("超时了...")
		case errors.Is(err, context.Canceled):
			fmt.Println("取消了...")
		default:
			// 别的错误
			fmt.Println(err)
		}
		return
	}
}

type Args struct {
	From    string
	Subject string
	Body    string
	IsHTML  bool
}

// testService 模拟实现 Sender[T any] 接口
type testService[T any] struct {
	err error
}

func (svc *testService[T]) Send(_ context.Context, _ string, _ T, _ ...string) error {
	return svc.err
}

```



## 连续超时故障转移

当连续出现 `context.DeadlineExceeded` 超时错误后会自动切换到下一个服务。同时还提供切换服务后，自动恢复原服务功能。

```go
package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/udugong/go-comm"
	"github.com/udugong/go-comm/failover"
)

func main() {
	// 当切换服务后触发30分钟后把服务切换回第一个
	setIdxFn := failover.WithSetIdxFunc[Args](func(ctx context.Context, idx *int32) {
		timer := time.NewTimer(30 * time.Minute)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			atomic.StoreInt32(idx, 0)
		}
	})
	// 创建一个连续3次超时就切换到下一个服务的发送服务。并设置定时恢复原服务。
	sender := failover.NewTimeoutService[Args]([]comm.Sender[Args]{&testService[Args]{}}, 3, setIdxFn)

	err := sender.Send(context.Background(), "", Args{}, "")
	if err != nil {
		// 只有非  context.DeadlineExceeded 的错误才会返回
		fmt.Println(err)
	}
}

type Args struct {
	From    string
	Subject string
	Body    string
	IsHTML  bool
}

// testService 模拟实现 Sender[T any] 接口
type testService[T any] struct {
	err error
}

func (svc *testService[T]) Send(_ context.Context, _ string, _ T, _ ...string) error {
	return svc.err
}

```

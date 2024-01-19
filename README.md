# go-comm
高可用的通信服务策略(短信、邮件)

使用泛型实现通用的发送服务接口，可适配多种服务商，并提供多种策略。

只需要根据服务商的规范实现 `Sender[T any]` 接口，选择所需要的策略，初始化时依赖注入即可。

```go
type Sender[T any] interface {
	// Send 发送信息
	// tpl 模板/模板id
	// args 参数
	// to 接收人
	Send(ctx context.Context, tpl string, args T, to ...string) error
}

```



go versions
==================

`>=1.20`



# usage

下载安装：`go get github.com/udugong/go-comm`

以下方式可以考虑嵌套使用。

  * [重试](#retryable-package)
  * [限流](#ratelimit-package)
  * [故障转移](#failover-package)



# `retryable` package

该`retryable`包提供了出错重试策略。

导入`"github.com/udugong/go-comm/retryable"`

- [直接重试](##直接重试)
- [间隔重试](##间隔重试)



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

该`ratelimit`包提供了限流策略。



# `failover` package

该`failover`包提供了故障转移策略。


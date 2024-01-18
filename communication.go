package comm

import "context"

type Sender[T any] interface {
	// Send 发送信息
	// tpl 模板/模板id
	// args 参数
	// to 接收人
	Send(ctx context.Context, tpl string, args T, to ...string) error
}

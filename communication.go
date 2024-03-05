package comm

import "context"

type Sender[T any] interface {
	// Send 发送信息
	// biz 含糊的业务,可以是模板/模板id
	// args 参数
	// to 接收人
	Send(ctx context.Context, biz string, args T, to ...string) error
}

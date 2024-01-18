package failover

import "errors"

var (
	ErrAllServiceFailed = errors.New("全部服务商都失败了")
)

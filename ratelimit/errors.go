package ratelimit

import "errors"

var (
	ErrLimited = errors.New("触发了限流")
)

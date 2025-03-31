package curl

import (
	"github.com/avast/retry-go/v4"
	"time"
)

type RetryPolicy struct {
	RetryCondFunc func(resp *Response) error //条件判断的方法，处理比较复杂的问题

	Attempts  uint                //最大重试次数
	Delay     time.Duration       //初始基础间隔
	MaxJitter time.Duration       //最大抖动间隔
	DelayType retry.DelayTypeFunc //指数退避 + 随机抖动
}

func (r *RetryPolicy) getRetryOptions() []retry.Option {
	opts := make([]retry.Option, 0)
	if r.Attempts == 0 {
		return opts
	}
	opts = append(opts, retry.Attempts(r.Attempts))
	if r.Delay > 0 {
		opts = append(opts, retry.Delay(r.Delay))
	}
	if r.MaxJitter > 0 {
		opts = append(opts, retry.MaxJitter(r.MaxJitter))
	}
	if r.DelayType != nil {
		opts = append(opts, retry.DelayType(r.DelayType))
	} else {
		if r.Delay > 0 {
			opts = append(opts, retry.DelayType(retry.FixedDelay))
		} else if r.MaxJitter > 0 {
			opts = append(opts, retry.DelayType(retry.RandomDelay))
		} else {
			opts = append(opts, retry.DelayType(retry.BackOffDelay))
		}
	}
	return opts
}

func (r *RetryPolicy) hasRetryError(retResp *Response) error {
	if r.Attempts == 0 {
		return nil
	}

	if r.RetryCondFunc != nil {
		return r.RetryCondFunc(retResp)
	}

	return nil
}

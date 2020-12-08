package backoff

import "time"

// An Operation is executing by Retry() or RetryNotify().
// The operation will be retried using a backoff policy if it returns an error.
type Operation func() error

// Notify is a notify-on-error function. It receives an operation error and
// backoff delay if the operation failed (with an error).
//
// NOTE that if the backoff policy stated to stop retrying,
// the notify function isn't called.
// 重试的通知函数
type Notify func(error, time.Duration)

// Retry the operation o until it does not return error or BackOff stops.
// o is guaranteed to be run at least once.
// It is the caller's responsibility to reset b after Retry returns.
//
// If o returns a *PermanentError, the operation is not retried, and the
// wrapped error is returned.
//
// Retry sleeps the goroutine for the duration returned by BackOff after a
// failed operation returns.
// 不带回调通知的重试
func Retry(o Operation, b BackOff) error { return RetryNotify(o, b, nil) }

// RetryNotify calls notify function with the error and wait duration
// for each failed attempt before sleep.
// 带回调通知的重试
func RetryNotify(operation Operation, b BackOff, notify Notify) error {
	var err error
	var next time.Duration

	// 转换backOff
	cb := ensureContext(b)

	// 重置backOff
	b.Reset()
	for {
		// 第一次直接执行
		if err = operation(); err == nil {
			return nil
		}

		// 业务抛出不重试的err
		if permanent, ok := err.(*PermanentError); ok {
			return permanent.Err
		}

		// 获取下次重试的时间 不包含operation指定的时间
		if next = b.NextBackOff(); next == Stop {
			return err
		}

		// 回调函数
		if notify != nil {
			notify(err, next)
		}

		// 定时器延迟重试
		t := time.NewTimer(next)

		select {
		case <-cb.Context().Done():
			t.Stop()
			return err
		case <-t.C:
		}
	}
}

// PermanentError signals that the operation should not be retried.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

// Permanent wraps the given err in a *PermanentError.
func Permanent(err error) *PermanentError {
	return &PermanentError{
		Err: err,
	}
}

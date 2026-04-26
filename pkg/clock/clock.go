package clock

import "time"

// Clock 定义当前时间来源，便于解析器在测试中稳定控制“现在”。
type Clock interface {
	// Now 返回当前时间。
	Now() time.Time
}

// SystemClock 使用系统时间作为当前时间来源。
type SystemClock struct{}

// Now 返回系统当前时间。
func (SystemClock) Now() time.Time {
	return time.Now()
}

package scheduler

import (
	"fmt"
	"strings"
	"time"

	"code88reset/pkg/logger"
)

type logAggregator struct {
	label     string
	interval  time.Duration
	buffer    []string
	lastFlush time.Time
}

func newLogAggregator(label string, interval time.Duration) *logAggregator {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &logAggregator{
		label:     label,
		interval:  interval,
		lastFlush: time.Now(),
	}
}

func (l *logAggregator) Add(format string, args ...interface{}) {
	if l == nil {
		return
	}

	l.buffer = append(l.buffer, fmt.Sprintf(format, args...))

	if time.Since(l.lastFlush) >= l.interval {
		l.Flush()
	}
}

func (l *logAggregator) Flush() {
	if l == nil || len(l.buffer) == 0 {
		l.lastFlush = time.Now()
		return
	}

	var (
		firstCheck string
		lastCheck  string
		otherMsgs  []string
	)

	for _, entry := range l.buffer {
		if strings.HasPrefix(entry, "检查时间:") {
			ts := strings.TrimSpace(strings.TrimPrefix(entry, "检查时间:"))
			if ts != "" {
				if firstCheck == "" {
					firstCheck = ts
				}
				lastCheck = ts
				continue
			}
		}
		otherMsgs = append(otherMsgs, entry)
	}

	minutes := int(l.interval.Minutes())
	if minutes == 0 {
		minutes = 1
	}

	switch {
	case firstCheck != "" && lastCheck != "" && firstCheck != lastCheck:
		logger.Info("[%s] 最近%d分钟检查: 已覆盖 %s 至 %s", l.label, minutes, firstCheck, lastCheck)
	case firstCheck != "":
		logger.Info("[%s] 最近%d分钟检查: 已于 %s 完成", l.label, minutes, firstCheck)
	}

	if len(otherMsgs) > 0 {
		logger.Info("[%s] 其他信息: %s", l.label, strings.Join(otherMsgs, "; "))
	}

	l.buffer = nil
	l.lastFlush = time.Now()
}

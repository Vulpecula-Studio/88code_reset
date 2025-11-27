package scheduler

import (
	"context"
	"time"
)

// loopController manages the shared ticking logic for schedulers.
type loopController struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	lastSubscriptionSlot time.Time
	subscriptionInterval time.Duration
}

func newLoopController(subscriptionInterval time.Duration) *loopController {
	ctx, cancel := context.WithCancel(context.Background())
	return &loopController{
		ctx:                  ctx,
		cancel:               cancel,
		subscriptionInterval: subscriptionInterval,
	}
}

// run starts the subscription-and-reset loop until Stop is called.
func (l *loopController) run(subscriptionCheck func(), resetCheck func()) {
	interval := l.subscriptionInterval
	if interval <= 0 {
		interval = time.Minute
	}
	alignSlot := func(t time.Time) time.Time {
		return t.Truncate(interval)
	}

	// 初始检查：确保启动时立即执行一次
	subscriptionCheck()
	l.lastSubscriptionSlot = alignSlot(time.Now())
	resetCheck()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			slot := alignSlot(now)
			if slot.After(l.lastSubscriptionSlot) {
				subscriptionCheck()
				l.lastSubscriptionSlot = slot
			}
			resetCheck()
		}
	}
}

func (l *loopController) Stop() {
	l.cancel()
}

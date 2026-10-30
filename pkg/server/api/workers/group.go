package workers

import (
	"context"
	"sync"
	"time"
)

type Group struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewGroup(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{cancel: cancel}, ctx
}

func (g *Group) Go(run func()) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		run()
	}()
}

func (g *Group) Stop() {
	g.cancel()
	g.wg.Wait()
}

func every(ctx context.Context, interval func() time.Duration, run func()) {
	for {
		run()
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval()):
		}
	}
}

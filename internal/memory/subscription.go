package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"libx.net/eventstream"
)

// subscription 内存模式订阅实现
type subscription struct {
	id      string
	topic   string
	handler eventstream.EventHandler
	config  *eventstream.SubscribeConfig
	active  int32 // 使用原子操作
	stopCh  chan struct{}
	eventCh chan *eventstream.Event
	wg      sync.WaitGroup
}

// newSubscription 创建新订阅
func newSubscription(topic string, handler eventstream.EventHandler, config *eventstream.SubscribeConfig) *subscription {
	return &subscription{
		id:      generateSubscriptionID(),
		topic:   topic,
		handler: handler,
		config:  config,
		active:  1,
		stopCh:  make(chan struct{}),
		eventCh: make(chan *eventstream.Event, config.BufferSize),
	}
}

// Topic 实现Subscription接口
func (s *subscription) Topic() string {
	return s.topic
}

// ID 实现Subscription接口
func (s *subscription) ID() string {
	return s.id
}

// IsActive 实现Subscription接口
func (s *subscription) IsActive() bool {
	return atomic.LoadInt32(&s.active) == 1
}

// start 启动订阅处理
func (s *subscription) start(ctx context.Context, pool eventstream.PoolSubmitter) {
	s.wg.Add(1)

	// 启动事件处理协程
	for i := 0; i < s.config.Concurrency; i++ {
		pool.Submit(func() {
			defer s.wg.Done()
			s.processEvents(ctx)
		})
	}
}

// stop 停止订阅
func (s *subscription) stop() {
	if atomic.CompareAndSwapInt32(&s.active, 1, 0) {
		close(s.stopCh)
		s.wg.Wait()
		close(s.eventCh)
	}
}

// deliver 投递事件到订阅
func (s *subscription) deliver(event *eventstream.Event) bool {
	if !s.IsActive() {
		return false
	}

	select {
	case s.eventCh <- event:
		return true
	case <-s.stopCh:
		return false
	default:
		// 缓冲区满，丢弃事件
		return false
	}
}

// processEvents 处理事件
func (s *subscription) processEvents(ctx context.Context) {
	for {
		select {
		case event, ok := <-s.eventCh:
			if !ok {
				return
			}
			s.handleEvent(ctx, event)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// handleEvent 处理单个事件
func (s *subscription) handleEvent(ctx context.Context, event *eventstream.Event) {
	var err error

	// 重试逻辑
	for attempt := 0; attempt <= s.config.RetryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			// 计算退避延迟
			delay := calculateBackoff(s.config.RetryPolicy, attempt-1)
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-time.After(delay):
			}
		}

		// 执行事件处理器
		err = s.handler(ctx, event)
		if err == nil {
			return // 处理成功
		}

		// 记录错误日志
		// TODO: 使用结构化日志
		if attempt < s.config.RetryPolicy.MaxRetries {
			// log.Printf("Event processing attempt %d failed: %v", attempt+1, err)
		}
	}

	// 所有重试都失败了
	// TODO: 发送到死信队列或记录失败日志
	// log.Printf("Event processing failed after %d attempts: %v", s.config.RetryPolicy.MaxRetries+1, err)
}

// subscriberManager 订阅管理器
type subscriberManager struct {
	subscribers map[string][]*subscription
	mu          sync.RWMutex
}

// newSubscriberManager 创建订阅管理器
func newSubscriberManager() *subscriberManager {
	return &subscriberManager{
		subscribers: make(map[string][]*subscription),
	}
}

// add 添加订阅
func (sm *subscriberManager) add(sub *subscription) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.subscribers[sub.topic] = append(sm.subscribers[sub.topic], sub)
}

// remove 移除订阅
func (sm *subscriberManager) remove(sub *subscription) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	subs := sm.subscribers[sub.topic]
	for i, s := range subs {
		if s.id == sub.id {
			// 从切片中移除
			sm.subscribers[sub.topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	// 如果主题没有订阅者了，删除主题
	if len(sm.subscribers[sub.topic]) == 0 {
		delete(sm.subscribers, sub.topic)
	}
}

// getSubscribers 获取主题的所有订阅者
func (sm *subscriberManager) getSubscribers(topic string) []*subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subs := sm.subscribers[topic]
	if len(subs) == 0 {
		return nil
	}

	// 返回副本，避免并发问题
	result := make([]*subscription, len(subs))
	copy(result, subs)
	return result
}

// getAllSubscribers 获取所有订阅者
func (sm *subscriberManager) getAllSubscribers() []*subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []*subscription
	for _, subs := range sm.subscribers {
		result = append(result, subs...)
	}
	return result
}

// count 获取订阅者数量
func (sm *subscriberManager) count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, subs := range sm.subscribers {
		count += len(subs)
	}
	return count
}

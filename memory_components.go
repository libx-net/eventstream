package eventstream

import (
	"sync"
)

// eventHistory 事件历史记录
type eventHistory struct {
	events  []*Event
	maxSize int
	mu      sync.RWMutex
}

// newEventHistory 创建事件历史记录
func newEventHistory(maxSize int) *eventHistory {
	return &eventHistory{
		events:  make([]*Event, 0, maxSize),
		maxSize: maxSize,
	}
}

// add 添加事件到历史记录
func (eh *eventHistory) add(event *Event) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	// 如果达到最大容量，移除最旧的事件
	if len(eh.events) >= eh.maxSize {
		eh.events = eh.events[1:]
	}

	eh.events = append(eh.events, event)
}

// get 获取历史事件
func (eh *eventHistory) get(limit int) []*Event {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	if limit <= 0 || limit > len(eh.events) {
		limit = len(eh.events)
	}

	// 返回最新的limit个事件
	start := len(eh.events) - limit
	result := make([]*Event, limit)
	copy(result, eh.events[start:])

	return result
}

// count 获取历史事件数量
func (eh *eventHistory) count() int {
	eh.mu.RLock()
	defer eh.mu.RUnlock()
	return len(eh.events)
}

// eventMetrics 事件指标
type eventMetrics struct {
	emitted    map[string]int64
	delivered  map[string]int64
	subscribed map[string]int64
	mu         sync.RWMutex
}

// newEventMetrics 创建事件指标
func newEventMetrics() *eventMetrics {
	return &eventMetrics{
		emitted:    make(map[string]int64),
		delivered:  make(map[string]int64),
		subscribed: make(map[string]int64),
	}
}

// incEmitted 增加发布计数
func (em *eventMetrics) incEmitted(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.emitted[topic]++
}

// incDelivered 增加投递计数
func (em *eventMetrics) incDelivered(topic string, count int) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.delivered[topic] += int64(count)
}

// incSubscribed 增加订阅计数
func (em *eventMetrics) incSubscribed(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.subscribed[topic]++
}

// decSubscribed 减少订阅计数
func (em *eventMetrics) decSubscribed(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.subscribed[topic] > 0 {
		em.subscribed[topic]--
	}
}

// getStats 获取统计信息
func (em *eventMetrics) getStats() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return map[string]interface{}{
		"emitted":    em.copyMap(em.emitted),
		"delivered":  em.copyMap(em.delivered),
		"subscribed": em.copyMap(em.subscribed),
	}
}

// copyMap 复制map
func (em *eventMetrics) copyMap(m map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

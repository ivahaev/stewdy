package stewdy

import "sync"

var (
	queues       = map[string]queue{}
	queuesLocker sync.RWMutex
)

type queue struct {
	id     string
	free   int
	queued int
}

func SetQueueStat(id string, free, queued int) {
	queuesLocker.Lock()
	queuesLocker.Unlock()

	queues[id] = queue{id: id, free: free, queued: queued}
}

func queueStat(id string) (free, queued int) {
	queuesLocker.RLock()
	queuesLocker.RUnlock()

	q, ok := queues[id]
	if !ok {
		return
	}

	return q.free, q.queued
}

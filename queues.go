package stewdy

import "sync"

var (
	queues       = map[string]queue{}
	queuesLocker sync.RWMutex
)

type queue struct {
	id          string
	free        int
	queued      int
	originating int
	connected   int
}

// SetQueueStat updates internal statistic of call queue.
// Takes queue id, number of free operators and number of queued calls as arguments.
func SetQueueStat(id string, free, queued int) {
	queuesLocker.Lock()
	defer queuesLocker.Unlock()

	queues[id] = queue{id: id, free: free, queued: queued}
}

func queueStat(id string) (free, queued int) {
	queuesLocker.RLock()
	defer queuesLocker.RUnlock()

	q := queues[id]

	return q.free, q.queued
}

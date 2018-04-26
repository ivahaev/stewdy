package stewdy

import "testing"

func TestSetQueueStat(t *testing.T) {
	queues = map[string]queue{}

	SetQueueStat("1", 1, 2)
	if len(queues) != 1 {
		t.Fatal("emtpty queues")
	}

	f, q := queueStat("1")
	if f != 1 {
		t.Fatalf("queueStat().free => %d, expected 1", f)
	}

	if q != 2 {
		t.Fatalf("queueStat().queued => %d, expected 2", q)
	}

	SetQueueStat("1", 10, 20)
	if len(queues) != 1 {
		t.Fatal("emtpty queues")
	}

	f, q = queueStat("1")
	if f != 10 {
		t.Fatalf("queueStat().free => %d, expected 10", f)
	}

	if q != 20 {
		t.Fatalf("queueStat().queued => %d, expected 20", q)
	}

	SetQueueStat("10", 100, 200)
	if len(queues) != 2 {
		t.Fatalf("len(queues) => %d, expected 2", len(queues))
	}

	free, queued := queueStat("10")
	if free != 100 {
		t.Errorf(`queueStat("10").free = %d, expected 100`, free)
	}
	if queued != 200 {
		t.Errorf(`queueStat("10").queued = %d, expected 200`, queued)
	}

	free, queued = queueStat("noExistingQ")
	if free != 0 {
		t.Errorf(`queueStat("noExistingQ").free = %d, expected 0`, free)
	}
	if queued != 0 {
		t.Errorf(`queueStat("noExistingQ").queued = %d, expected 0`, queued)
	}
}

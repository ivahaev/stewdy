package stewdy

import "testing"

func TestSetQueueStat(t *testing.T) {
	if len(queues) != 0 {
		t.Fatal("not emtpty queues")
	}

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
}

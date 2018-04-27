package stewdy

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestUpdateCampaign(t *testing.T) {
	testData := []struct {
		c   Campaign
		err error
	}{
		{
			c: Campaign{
				Id:               "1",
				QueueID:          "Q1",
				IsActive:         true,
				MaxAttempts:      3,
				NextAttemptDelay: 60,
				ConcurrentCalls:  10,
				WaitForAnswer:    60,
				WaitForConnect:   600,
				MaxCallDuration:  3600,
				TimeTable: []*Schedule{
					{
						Id:      "S1",
						Weekday: 1,
						Start:   600,
						Stop:    900,
					},
					{
						Id:      "S2",
						Weekday: 2,
						Start:   610,
						Stop:    910,
					},
				},
			},
			err: nil,
		},
		{
			c: Campaign{
				Id:               "1",
				QueueID:          "",
				IsActive:         true,
				MaxAttempts:      3,
				NextAttemptDelay: 60,
				ConcurrentCalls:  10,
				TimeTable: []*Schedule{
					{
						Id:      "S1",
						Weekday: 1,
						Start:   600,
						Stop:    900,
					},
					{
						Id:      "S2",
						Weekday: 2,
						Start:   610,
						Stop:    910,
					},
				},
			},
			err: ErrNoQID,
		},
		{
			c: Campaign{
				Id:               "1",
				QueueID:          "Q1",
				IsActive:         true,
				MaxAttempts:      3,
				NextAttemptDelay: 60,
				ConcurrentCalls:  10,
				TimeTable: []*Schedule{
					{
						Id:      "S1",
						Weekday: 1,
						Start:   600,
						Stop:    900,
					},
					{
						Id:      "S2",
						Weekday: 1,
						Start:   610,
						Stop:    910,
					},
				},
			},
			err: errors.New("schedule ID S1 overlaps with ID S2"),
		},
		{
			c: Campaign{
				Id:               "1",
				QueueID:          "Q1",
				IsActive:         true,
				MaxAttempts:      3,
				NextAttemptDelay: 60,
				ConcurrentCalls:  10,
				TimeTable:        []*Schedule{},
			},
			err: ErrEmptyTimeTable,
		},
		{
			c: Campaign{
				Id:               "1",
				QueueID:          "Q1",
				IsActive:         true,
				MaxAttempts:      3,
				NextAttemptDelay: 60,
				ConcurrentCalls:  10,
				TimeTable: []*Schedule{
					{
						Id:      "S1",
						Weekday: 1,
						Start:   600,
						Stop:    1500,
					},
				},
			},
			err: errors.New("invalid time range in schedule ID S1"),
		},
	}

	for i, d := range testData {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			err := UpdateCampaign(d.c)
			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", d.err) {
				t.Errorf("UpdateCampaign(d.c) => %v, expected %v", err, d.err)
			}

			if err != nil {
				return
			}

			cmp, err := db.getCampaign(d.c.GetId())
			if err != nil {
				t.Errorf("db.getCampaign(%s) => unexpected error: %v", d.c.GetId(), err)
			}

			if cmp.GetId() != d.c.GetId() {
				t.Errorf("cmp.Id => %s, expected %s", cmp.GetId(), d.c.GetId())
			}

			if cmp.GetQueueID() != d.c.GetQueueID() {
				t.Errorf("cmp.QueueID => %s, expected %s", cmp.GetQueueID(), d.c.GetQueueID())
			}
		})
	}
}

func TestOverlaping(t *testing.T) {
	testData := []struct {
		s1  *Schedule
		s2  *Schedule
		res bool
	}{
		{
			s1:  &Schedule{Weekday: 1, Start: 600, Stop: 1200},
			s2:  &Schedule{Weekday: 1, Start: 900, Stop: 1260},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 1, Start: 900, Stop: 1260},
			s2:  &Schedule{Weekday: 1, Start: 600, Stop: 1200},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 1, Start: 600, Stop: 1200},
			s2:  &Schedule{Weekday: 1, Start: 900, Stop: 1000},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 1, Start: 900, Stop: 1000},
			s2:  &Schedule{Weekday: 1, Start: 600, Stop: 1200},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 1, Start: 600, Stop: 1200},
			s2:  &Schedule{Weekday: 5, Start: 900, Stop: 1260},
			res: false,
		},
		{
			s1:  &Schedule{Weekday: 1, Start: 600, Stop: 800},
			s2:  &Schedule{Weekday: 1, Start: 900, Stop: 1260},
			res: false,
		},
		{
			s1:  &Schedule{Weekday: 1, Start: 800, Stop: 600},
			s2:  &Schedule{Weekday: 2, Start: 400, Stop: 1260},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 1, Start: 800, Stop: 300},
			s2:  &Schedule{Weekday: 2, Start: 400, Stop: 1260},
			res: false,
		},
		{
			s1:  &Schedule{Weekday: 2, Start: 400, Stop: 1260},
			s2:  &Schedule{Weekday: 1, Start: 800, Stop: 600},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 6, Start: 800, Stop: 600},
			s2:  &Schedule{Weekday: 0, Start: 400, Stop: 1260},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 6, Start: 800, Stop: 600},
			s2:  &Schedule{Weekday: 0, Start: 400, Stop: 500},
			res: true,
		},
		{
			s1:  &Schedule{Weekday: 0, Start: 400, Stop: 500},
			s2:  &Schedule{Weekday: 6, Start: 800, Stop: 600},
			res: true,
		},
	}

	for i, v := range testData {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			res := isOverlaped(v.s1, v.s2)
			if res != v.res {
				t.Errorf("isOverlaped(v.s1, v.s2) => %t, expected %t", res, v.res)
			}
		})
	}
}

func TestAddTargets(t *testing.T) {
	c := Campaign{
		Id:      "Add1",
		QueueID: "QAdd1",
		TimeTable: []*Schedule{
			{
				Id:    "1",
				Start: 10,
				Stop:  20,
			},
		},
	}

	err := UpdateCampaign(c)
	if err != nil {
		t.Fatal(err)
	}

	targets := []*Target{
		{
			Id:          "T1",
			PhoneNumber: "123",
		},
		{
			Id:          "T2",
			PhoneNumber: "234",
		},
	}

	err = AddTargets(c.Id, targets)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCalcBatchSize(t *testing.T) {
	testData := []struct {
		c   *campaign
		max int32
		res int32
	}{
		{
			c:   &campaign{currentBatchSize: 0},
			max: 10,
			res: 2,
		},
		{
			c:   &campaign{currentBatchSize: 0},
			max: 1,
			res: 1,
		},
		{
			c:   &campaign{currentBatchSize: 1},
			max: 10,
			res: 2,
		},
		{
			c:   &campaign{currentBatchSize: 10},
			max: 12,
			res: 12,
		},
		{
			c:   &campaign{currentBatchSize: 4},
			max: 12,
			res: 8,
		},
		{
			c:   &campaign{currentBatchSize: 4},
			max: 2,
			res: 2,
		},
		{
			c:   &campaign{currentBatchSize: 3},
			max: 9,
			res: 6,
		},
	}

	for i, v := range testData {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			c := v.c
			currentBatchSize := c.currentBatchSize
			res := c.calcBatchSize(v.max)
			if res != v.res {
				t.Errorf("currentBatchSize was %d, c.calcBatchSize(%d) = %d, expected %d", currentBatchSize, v.max, c.currentBatchSize, v.res)
			}
		})
	}
}

func TestFreeSlots(t *testing.T) {
	testData := []struct {
		name         string
		q            string
		free, queued int
		c            *campaign
		res          int32
	}{
		{
			name:   "1",
			q:      "testQ",
			free:   10,
			queued: 0,
			c: &campaign{
				currentBatchSize: 2,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 40,
					BatchSize:       20,
				},
				m: &sync.Mutex{},
			},
			res: 4,
		},
		{
			name:   "2",
			q:      "testQ",
			free:   30,
			queued: 0,
			c: &campaign{
				currentBatchSize: 20,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 40,
					BatchSize:       20,
				},
				m: &sync.Mutex{},
			},
			res: 20,
		},
		{
			name:   "3",
			q:      "testQ",
			free:   10,
			queued: 10,
			c: &campaign{
				currentBatchSize: 20,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 40,
					BatchSize:       20,
				},
				m: &sync.Mutex{},
			},
			res: 0,
		},
		{
			name:   "4",
			q:      "testQ",
			free:   3,
			queued: 0,
			c: &campaign{
				currentBatchSize: 20,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 40,
					BatchSize:       20,
				},
				m: &sync.Mutex{},
			},
			res: 3,
		},
		{
			name:   "5",
			q:      "testQ",
			free:   30,
			queued: 7,
			c: &campaign{
				currentBatchSize: 9,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 40,
					BatchSize:       20,
				},
				m: &sync.Mutex{},
			},
			res: 18,
		},
		{
			name:   "6",
			q:      "testQ",
			free:   30,
			queued: 7,
			c: &campaign{
				currentBatchSize: 9,
				c: Campaign{
					QueueID:         "unknownQ",
					ConcurrentCalls: 40,
					BatchSize:       20,
				},
				m: &sync.Mutex{},
			},
			res: 0,
		},
		{
			name:   "7",
			q:      "testQ",
			free:   30,
			queued: 7,
			c: &campaign{
				currentBatchSize: 9,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 40,
					BatchSize:       20,
				},
				m:           &sync.Mutex{},
				originating: map[string]*Target{"1": &Target{}, "2": &Target{}},
			},
			res: 18,
		},
		{
			name:   "8",
			q:      "testQ",
			free:   30,
			queued: 7,
			c: &campaign{
				currentBatchSize: 10,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 20,
					BatchSize:       20,
				},
				m:           &sync.Mutex{},
				originating: map[string]*Target{"1": &Target{}, "2": &Target{}},
			},
			res: 18,
		},
		{
			name:   "9",
			q:      "testQ",
			free:   30,
			queued: 7,
			c: &campaign{
				currentBatchSize: 10,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 20,
					BatchSize:       20,
				},
				m:           &sync.Mutex{},
				originating: map[string]*Target{"1": &Target{}, "2": &Target{}},
				answered:    map[string]*Target{"11": &Target{}},
			},
			res: 17,
		},
		{
			name:   "10",
			q:      "testQ",
			free:   30,
			queued: 7,
			c: &campaign{
				currentBatchSize: 10,
				c: Campaign{
					QueueID:         "testQ",
					ConcurrentCalls: 20,
					BatchSize:       20,
				},
				m:           &sync.Mutex{},
				originating: map[string]*Target{"1": &Target{}, "2": &Target{}},
				answered:    map[string]*Target{"11": &Target{}},
				connected:   map[string]*Target{"22": &Target{}, "33": &Target{}, "44": &Target{}},
			},
			res: 14,
		},
	}

	for _, v := range testData {
		t.Run(v.name, func(t *testing.T) {
			SetQueueStat(v.q, v.free, v.queued)
			res := v.c.freeSlots()
			if res != v.res {
				t.Errorf("c.freeSlots() => %d, expected %d", res, v.res)
			}
		})
	}
}

func TestClearTargets(t *testing.T) {
	now := time.Now()

	testData := []struct {
		name                                      string
		c                                         *campaign
		originatingCnt, answeredCnt, connectedCnt int
		failedTargets                             map[string]struct{}
		hangupedTargets                           map[string]struct{}
	}{
		{
			name: "1",
			c: &campaign{
				waitForAnswer:  time.Minute,
				waitForConnect: 10 * time.Minute,
				waitForHangup:  time.Hour,
				c:              Campaign{Id: "C1", MaxAttempts: 2},
				l:              list.New(),
				originating: map[string]*Target{
					"o111": &Target{Id: "o111", LastAttemptTime: now.Unix(), Attempts: 1},
					"o222": &Target{Id: "o222", LastAttemptTime: now.Unix(), Attempts: 1},
					"o333": &Target{Id: "o333", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"o444": &Target{Id: "o444", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 2},
				},
				answered: map[string]*Target{
					"a111": &Target{Id: "a111", AnswerTime: now.Unix(), Attempts: 1},
					"a222": &Target{Id: "a222", AnswerTime: now.Unix(), Attempts: 1},
					"a333": &Target{Id: "a333", AnswerTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"a444": &Target{Id: "a444", AnswerTime: now.Add(-2 * time.Minute).Unix(), Attempts: 2},
				},
				connected: map[string]*Target{
					"c111": &Target{Id: "c111", ConnectTime: now.Unix(), Attempts: 1},
					"c222": &Target{Id: "c222", ConnectTime: now.Unix(), Attempts: 1},
					"c333": &Target{Id: "c333", ConnectTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"c444": &Target{Id: "c444", ConnectTime: now.Add(-2 * time.Minute).Unix(), Attempts: 2},
				},
				m: &sync.Mutex{},
			},
			originatingCnt: 2,
			answeredCnt:    4,
			connectedCnt:   4,
			failedTargets:  map[string]struct{}{"o444": struct{}{}},
		},
		{
			name: "2",
			c: &campaign{
				waitForAnswer:  time.Minute,
				waitForConnect: 10 * time.Minute,
				waitForHangup:  time.Hour,
				c:              Campaign{Id: "C1", MaxAttempts: 2},
				l:              list.New(),
				originating: map[string]*Target{
					"o111": &Target{Id: "o111", LastAttemptTime: now.Unix(), Attempts: 1},
					"o222": &Target{Id: "o222", LastAttemptTime: now.Unix(), Attempts: 1},
					"o333": &Target{Id: "o333", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"o444": &Target{Id: "o444", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
				},
				answered: map[string]*Target{
					"a111": &Target{Id: "a111", AnswerTime: now.Unix(), Attempts: 1},
					"a222": &Target{Id: "a222", AnswerTime: now.Unix(), Attempts: 1},
					"a333": &Target{Id: "a333", AnswerTime: now.Add(-20 * time.Minute).Unix(), Attempts: 1},
					"a444": &Target{Id: "a444", AnswerTime: now.Add(-2 * time.Minute).Unix(), Attempts: 2},
				},
				connected: map[string]*Target{
					"c111": &Target{Id: "c111", ConnectTime: now.Unix(), Attempts: 1},
					"c222": &Target{Id: "c222", ConnectTime: now.Unix(), Attempts: 1},
					"c333": &Target{Id: "c333", ConnectTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"c444": &Target{Id: "c444", ConnectTime: now.Add(-2 * time.Minute).Unix(), Attempts: 2},
				},
				m: &sync.Mutex{},
			},
			originatingCnt: 2,
			answeredCnt:    3,
			connectedCnt:   4,
			failedTargets:  map[string]struct{}{},
		},
		{
			name: "3",
			c: &campaign{
				waitForAnswer:  time.Minute,
				waitForConnect: 10 * time.Minute,
				waitForHangup:  time.Hour,
				c:              Campaign{Id: "C1", MaxAttempts: 2},
				l:              list.New(),
				originating: map[string]*Target{
					"o111": &Target{Id: "o111", LastAttemptTime: now.Unix(), Attempts: 1},
					"o222": &Target{Id: "o222", LastAttemptTime: now.Unix(), Attempts: 1},
					"o333": &Target{Id: "o333", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"o444": &Target{Id: "o444", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
				},
				answered: map[string]*Target{
					"a111": &Target{Id: "a111", AnswerTime: now.Unix(), Attempts: 1},
					"a222": &Target{Id: "a222", AnswerTime: now.Unix(), Attempts: 1},
					"a333": &Target{Id: "a333", AnswerTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"a444": &Target{Id: "a444", AnswerTime: now.Add(-2 * time.Minute).Unix(), Attempts: 2},
				},
				connected: map[string]*Target{
					"c111": &Target{Id: "c111", ConnectTime: now.Unix(), Attempts: 1},
					"c222": &Target{Id: "c222", ConnectTime: now.Unix(), Attempts: 1},
					"c333": &Target{Id: "c333", ConnectTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"c444": &Target{Id: "c444", ConnectTime: now.Add(-2 * time.Hour).Unix(), Attempts: 2},
				},
				m: &sync.Mutex{},
			},
			originatingCnt:  2,
			answeredCnt:     4,
			connectedCnt:    3,
			failedTargets:   map[string]struct{}{},
			hangupedTargets: map[string]struct{}{"c444": struct{}{}},
		},
		{
			name: "4",
			c: &campaign{
				waitForAnswer:  time.Minute,
				waitForConnect: 10 * time.Minute,
				waitForHangup:  time.Hour,
				c:              Campaign{Id: "C1", MaxAttempts: 2},
				l:              list.New(),
				originating: map[string]*Target{
					"o111": &Target{Id: "o111", LastAttemptTime: now.Unix(), Attempts: 1},
					"o222": &Target{Id: "o222", LastAttemptTime: now.Unix(), Attempts: 1},
					"o333": &Target{Id: "o333", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 1},
					"o444": &Target{Id: "o444", LastAttemptTime: now.Add(-2 * time.Minute).Unix(), Attempts: 2},
				},
				answered: map[string]*Target{
					"a111": &Target{Id: "a111", AnswerTime: now.Unix(), Attempts: 1},
					"a222": &Target{Id: "a222", AnswerTime: now.Unix(), Attempts: 1},
					"a333": &Target{Id: "a333", AnswerTime: now.Add(-20 * time.Minute).Unix(), Attempts: 1},
					"a444": &Target{Id: "a444", AnswerTime: now.Add(-20 * time.Minute).Unix(), Attempts: 2},
				},
				connected: map[string]*Target{
					"c111": &Target{Id: "c111", ConnectTime: now.Unix(), Attempts: 1},
					"c222": &Target{Id: "c222", ConnectTime: now.Unix(), Attempts: 1},
					"c333": &Target{Id: "c333", ConnectTime: now.Add(-2 * time.Hour).Unix(), Attempts: 1},
					"c444": &Target{Id: "c444", ConnectTime: now.Add(-2 * time.Hour).Unix(), Attempts: 2},
				},
				m: &sync.Mutex{},
			},
			originatingCnt:  2,
			answeredCnt:     2,
			connectedCnt:    2,
			failedTargets:   map[string]struct{}{"o444": struct{}{}, "a444": struct{}{}},
			hangupedTargets: map[string]struct{}{"c444": struct{}{}, "c333": struct{}{}},
		},
	}

	eventHandlers = map[TargetEvent][]EventHandler{}
	failedTargets := map[string]struct{}{}
	hangupedTargets := map[string]struct{}{}
	On(EventFail, func(t Target) {
		failedTargets[t.Id] = struct{}{}
	})
	On(EventHangup, func(t Target) {
		hangupedTargets[t.Id] = struct{}{}
	})

	for _, v := range testData {
		failedTargets = map[string]struct{}{}
		hangupedTargets = map[string]struct{}{}

		t.Run(v.name, func(t *testing.T) {
			c := v.c
			c.cleanTargets(now)
			if len(c.originating) != v.originatingCnt {
				t.Errorf("len(c.originating) = %d, expected %d", len(c.originating), v.originatingCnt)
			}
			if len(c.answered) != v.answeredCnt {
				t.Errorf("len(c.answered) = %d, expected %d", len(c.answered), v.answeredCnt)
			}
			if len(c.connected) != v.connectedCnt {
				t.Errorf("len(c.connected) = %d, expected %d", len(c.connected), v.connectedCnt)
			}
			if len(failedTargets) != len(v.failedTargets) {
				t.Fatalf("failedTargets = %v, expected %v", failedTargets, v.failedTargets)
			}
			for k := range failedTargets {
				if _, ok := v.failedTargets[k]; !ok {
					t.Errorf("Unexpected failed target Id: %s", k)
				}
			}
			if len(hangupedTargets) != len(v.hangupedTargets) {
				t.Fatalf("hangupedTargets = %v, expected %v", hangupedTargets, v.hangupedTargets)
			}
			for k := range hangupedTargets {
				if _, ok := v.hangupedTargets[k]; !ok {
					t.Errorf("Unexpected hanguped target Id: %s", k)
				}
			}
		})
	}

	eventHandlers = map[TargetEvent][]EventHandler{}
}

func TestNext(t *testing.T) {
	now := time.Now()
	testData := []struct {
		name                      string
		c                         *campaign
		targets                   []*Target
		count                     int32
		originatingLen, targetLen int
		res                       map[string]struct{}
	}{
		{
			name: "1",
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				m:               &sync.Mutex{},
			},
			count: 0,
			res:   map[string]struct{}{},
		},
		{
			name: "2",
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				m:               &sync.Mutex{},
			},
			count: 2,
			res:   map[string]struct{}{},
		},
		{
			name: "3",
			targets: []*Target{
				&Target{Id: "T1"},
			},
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				originating:     map[string]*Target{},
				c: Campaign{
					Id:          "C1",
					MaxAttempts: 3,
				},
				m: &sync.Mutex{},
			},
			count:          10,
			originatingLen: 1,
			targetLen:      0,
			res:            map[string]struct{}{"T1": struct{}{}},
		},
		{
			name: "4",
			targets: []*Target{
				&Target{Id: "T1"},
				&Target{Id: "T2"},
				&Target{Id: "T3"},
				&Target{Id: "T4"},
				&Target{Id: "T5"},
			},
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				originating:     map[string]*Target{},
				c: Campaign{
					Id:          "C1",
					MaxAttempts: 3,
				},
				m: &sync.Mutex{},
			},
			count:          2,
			originatingLen: 2,
			targetLen:      3,
			res:            map[string]struct{}{"T1": struct{}{}, "T2": struct{}{}},
		},
		{
			name: "5",
			targets: []*Target{
				&Target{Id: "T1", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T2"},
				&Target{Id: "T3"},
				&Target{Id: "T4"},
				&Target{Id: "T5"},
			},
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				originating:     map[string]*Target{},
				c: Campaign{
					Id:          "C1",
					MaxAttempts: 3,
				},
				m: &sync.Mutex{},
			},
			count:          2,
			originatingLen: 2,
			targetLen:      3,
			res:            map[string]struct{}{"T2": struct{}{}, "T3": struct{}{}},
		},
		{
			name: "6",
			targets: []*Target{
				&Target{Id: "T1", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T2"},
				&Target{Id: "T3", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T4"},
				&Target{Id: "T5"},
			},
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				originating:     map[string]*Target{},
				c: Campaign{
					Id:          "C1",
					MaxAttempts: 3,
				},
				m: &sync.Mutex{},
			},
			count:          2,
			originatingLen: 2,
			targetLen:      3,
			res:            map[string]struct{}{"T2": struct{}{}, "T4": struct{}{}},
		},
		{
			name: "7",
			targets: []*Target{
				&Target{Id: "T1", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T2"},
				&Target{Id: "T3", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T4", Attempts: 3},
				&Target{Id: "T5"},
			},
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				originating:     map[string]*Target{},
				c: Campaign{
					Id:          "C1",
					MaxAttempts: 3,
				},
				m: &sync.Mutex{},
			},
			count:          2,
			originatingLen: 2,
			targetLen:      2,
			res:            map[string]struct{}{"T2": struct{}{}, "T5": struct{}{}},
		},
		{
			name: "8",
			targets: []*Target{
				&Target{Id: "T1", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T2", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T3", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T4", Attempts: 3},
				&Target{Id: "T5", NextAttemptTime: now.Add(time.Hour).Unix()},
			},
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Unix(),
				originating:     map[string]*Target{},
				c: Campaign{
					Id:          "C1",
					MaxAttempts: 3,
				},
				m: &sync.Mutex{},
			},
			count:          2,
			originatingLen: 0,
			targetLen:      4,
			res:            map[string]struct{}{},
		},
		{
			name: "9",
			targets: []*Target{
				&Target{Id: "T1", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T2"},
				&Target{Id: "T3", NextAttemptTime: now.Add(time.Hour).Unix()},
				&Target{Id: "T4", Attempts: 3},
				&Target{Id: "T5"},
			},
			c: &campaign{
				l:               list.New(),
				nextAttemptTime: now.Add(time.Minute).Unix(),
				originating:     map[string]*Target{},
				c: Campaign{
					Id:          "C1",
					MaxAttempts: 3,
				},
				m: &sync.Mutex{},
			},
			count:          2,
			originatingLen: 0,
			targetLen:      5,
			res:            map[string]struct{}{},
		},
	}

	for _, v := range testData {
		t.Run(v.name, func(t *testing.T) {
			c := v.c
			c.addTargets(v.targets)
			res := c.next(v.count)
			if len(c.originating) != v.originatingLen {
				t.Errorf("len(c.originating) = %d, expected %d", len(c.originating), v.originatingLen)
			}
			if c.l.Len() != v.targetLen {
				t.Errorf("c.l.Len() = %d, expected %d", c.l.Len(), v.targetLen)
			}
			if len(res) != len(v.res) {
				t.Fatalf("targets count = %d, expected %d", len(res), len(v.res))
			}
			for _, trg := range res {
				if _, ok := v.res[trg.Id]; !ok {
					t.Errorf("Unexpected target Id: %s in c.nextAtTime(%d, %v)", trg.Id, v.count, now)
				}
			}
		})
	}
}

package stewdy

import (
	"errors"
	"fmt"
	"sync"
	"testing"
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

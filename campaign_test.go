package stewdy

import (
	"errors"
	"fmt"
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

package stewdy

import (
	"container/list"
	"errors"
	fmt "fmt"
	"math"
	"sort"
	"sync"
	"time"
)

var (
	campaigns  = map[string]campaign{}
	campaignsm sync.RWMutex
)

var (
	ErrNoQID          = errors.New("no queue id provided")
	ErrEmptyTimeTable = errors.New("timetable is empty")
)

type campaign struct {
	c               Campaign
	l               *list.List
	e               *list.Element
	nextAttemptTime int64
	active          map[string]*Target
	failedIn        time.Duration
	m               *sync.Mutex
}

func (c campaign) addTargets(targets []*Target) {
	c.m.Lock()
	for _, v := range targets {
		c.l.PushBack(v)
	}

	c.m.Unlock()
}

func (c campaign) next(n int) []*Target {
	return c.nextAtTime(n, time.Now())
}

func (c campaign) nextAtTime(n int, t time.Time) []*Target {
	c.m.Lock()
	defer c.m.Unlock()

	res := make([]*Target, 0, n)
	if n == 0 {
		return res
	}

	if c.l.Len() == 0 {
		return res
	}

	attemptTime := t.Unix()
	if c.nextAttemptTime > attemptTime {
		return res
	}

	if c.e == nil {
		c.sort()
		c.e = c.l.Front()
	}

	ids := make([]string, 0, n)
	defer func() {
		if len(ids) == 0 {
			return
		}

		go c.failedIDs(ids)
		err := db.saveMany(c.c.GetId(), res)
		if err != nil {
			panic(err)
		}
	}()

	for e := c.e; e != nil; e = e.Next() {
		c.l.Remove(e)
		t := e.Value.(*Target)
		ids = append(ids, t.GetId())
		if t.GetAttempts() >= c.c.GetMaxAttempts() {
			continue
		}

		t.Attempts++
		c.active[t.GetId()] = t
		res = append(res, t)
		if len(res) == n {
			return res
		}
	}

	return res
}

func (c campaign) failedIDs(ids []string) {
	time.Sleep(c.failedIn)
	c.m.Lock()
	defer c.m.Unlock()

	for _, id := range ids {
		t, ok := c.active[id]
		if !ok {
			continue
		}

		delete(c.active, id)
		if t.Attempts >= c.c.GetMaxAttempts() {
			go emit(Fail, *t)
			continue
		}

		t.NextAttemptTime = t.LastAttemptTime + c.c.NextAttemptDelay
		c.l.PushFront(t)
	}
}

// sort does not lock mutex so you should do it by yourself.
func (c campaign) sort() {
	targets := make([]*Target, 0, c.l.Len())
	for e := c.l.Front(); e != nil; e = e.Next() {
		t := e.Value.(*Target)
		targets = append(targets, t)
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].GetNextAttemptTime() < targets[j].GetNextAttemptTime()
	})

	c.l.Init()
	for _, v := range targets {
		c.l.PushBack(v)
		if nat := v.GetNextAttemptTime(); c.nextAttemptTime > nat {
			c.nextAttemptTime = nat
		}
	}
}

func validateTimetable(c Campaign) error {
	if len(c.TimeTable) == 0 {
		return ErrEmptyTimeTable
	}

	for _, s := range c.TimeTable {
		if s.Start < 0 || s.Stop > 1440 || s.Start == s.Stop {
			return fmt.Errorf("invalid time range in schedule ID %v", s.Id)
		}
	}

	for _, s1 := range c.TimeTable {
		for _, s2 := range c.TimeTable {
			if s1.GetId() == s2.GetId() {
				continue
			}

			if isOverlaped(s1, s2) {
				return fmt.Errorf("schedule ID %s overlaps with ID %s", s1.GetId(), s2.GetId())
			}
		}
	}

	return nil
}

func isOverlaped(s1, s2 *Schedule) bool {
	start1, stop1 := s1.Start, s1.Stop
	start2, stop2 := s2.Start, s2.Stop
	wd1, wd2 := s1.Weekday, s2.Weekday
	if wd1 == 0 {
		wd1 = 7
	}
	if wd2 == 0 {
		wd2 = 7
	}

	if math.Abs(float64(wd1-wd2)) > 1 {
		return false
	}

	if start1 > stop1 {
		stop1 += 1440
	}
	if start2 > stop2 {
		stop2 += 1440
	}

	switch {
	case wd1 < wd2:
		start2 += 1440
		stop2 += 1440
	case wd1 > wd2:
		start1 += 1440
		stop1 += 1440
		return isTimeOverlaped(start2, stop2, start1, stop1)
	}

	return isTimeOverlaped(start1, stop1, start2, stop2)
}

func isTimeOverlaped(start1, stop1, start2, stop2 int32) bool {
	if start1 < start2 {
		return stop1 > start2
	}

	return start1 < stop2
}

func UpdateCampaign(c Campaign) error {
	if len(c.QueueID) == 0 {
		return ErrNoQID
	}

	if err := validateTimetable(c); err != nil {
		return err
	}

	err := db.saveCampaign(c)
	if err != nil {
		return err
	}

	campaignsm.Lock()
	cmp, ok := campaigns[c.GetId()]
	if !ok {
		cmp = campaign{
			c:        c,
			l:        list.New(),
			active:   map[string]*Target{},
			failedIn: time.Minute,
			m:        &sync.Mutex{},
		}
	}
	campaignsm.Unlock()

	cmp.c = c

	return nil
}

func AddTargets(campaignID string, targets []*Target) error {
	campaignsm.Lock()
	c, ok := campaigns[campaignID]
	campaignsm.Unlock()
	if !ok {
		return ErrNotFound
	}

	for _, v := range targets {
		v.CampaignID = campaignID
		v.LastAttemptTime = 0
		v.Attempts = 0
		v.UniqueId = ""
	}

	err := db.saveMany(campaignID, targets)
	if err != nil {
		return err
	}

	c.addTargets(targets)

	return nil
}

func RemoveTarget(campaignID, targetID string) {
	campaignsm.Lock()
	defer campaignsm.Unlock()

	c, ok := campaigns[campaignID]
	if !ok {
		return
	}

	delete(c.active, targetID)

	for e := c.l.Front(); e != nil; e = e.Next() {
		t := e.Value.(*Target)
		if t.GetId() == targetID {
			c.l.Remove(e)
			db.delete(campaignID, targetID)
			return
		}
	}
}

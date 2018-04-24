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
	originating     map[string]*Target
	answered        map[string]*Target
	failedIn        time.Duration
	waitForConnect  time.Duration
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
		t.LastAttemptTime = time.Now().Unix()
		c.originating[t.GetId()] = t
		res = append(res, t)
		if len(res) == n {
			return res
		}
	}

	return res
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

func (c campaign) targetsCleaner() {
	ticker := time.NewTicker(c.failedIn)
	for range ticker.C {
		c.cleanTargets()
	}
}

func (c campaign) cleanTargets() {
	c.m.Lock()
	defer c.m.Unlock()

	failedTime := time.Now().Add(-1 * c.failedIn).Unix()
	notConnectedTime := time.Now().Add(-1 * c.waitForConnect).Unix()
	updated := []*Target{}
	deleted := []string{}

	for k, v := range c.originating {
		if v.LastAttemptTime <= failedTime {
			delete(c.originating, k)
			if v.Attempts >= c.c.GetMaxAttempts() {
				go emit(EventFail, *v)
				deleted = append(deleted, v.GetId())
				continue
			}

			v.NextAttemptTime = v.LastAttemptTime + c.c.NextAttemptDelay
			v.LastAttemptTime = 0
			c.l.PushFront(v)
			updated = append(updated, v)
		}
	}

	for k, v := range c.answered {
		if v.AnswerTime <= notConnectedTime {
			delete(c.answered, k)
			if v.Attempts >= c.c.GetMaxAttempts() {
				go emit(EventFail, *v)
				deleted = append(deleted, v.GetId())
				continue
			}

			v.NextAttemptTime = v.LastAttemptTime + c.c.NextAttemptDelay
			v.LastAttemptTime = 0
			c.l.PushFront(v)
			updated = append(updated, v)
		}
	}

	if len(updated) > 0 {
		err := db.saveMany(c.c.GetId(), updated)
		if err != nil {
			panic(err)
		}
	}

	if len(deleted) > 0 {
		db.deleteMany(c.c.GetId(), deleted)
	}
}

func answered(campaignID, targetID, uniqueID string) error {
	campaignsm.RLock()
	c, ok := campaigns[campaignID]
	campaignsm.RUnlock()
	if !ok {
		return ErrNotFound
	}

	c.m.Lock()
	defer c.m.Unlock()

	t, ok := c.originating[targetID]
	if !ok {
		return ErrNotFound
	}

	delete(c.originating, targetID)
	t.UniqueId = uniqueID
	go emit(EventAnswer, *t)
	c.answered[uniqueID] = t

	err := db.save(campaignID, t)
	if err != nil {
		panic(err)
	}

	return nil
}

func connected(campaignID, uniqueID, operatorID string) error {
	campaignsm.RLock()
	c, ok := campaigns[campaignID]
	campaignsm.RUnlock()
	if !ok {
		return ErrNotFound
	}

	c.m.Lock()
	defer c.m.Unlock()

	t, ok := c.answered[uniqueID]
	if !ok {
		return ErrNotFound
	}

	delete(c.answered, uniqueID)
	t.OperatorID = operatorID
	go emit(EventConnect, *t)
	err := db.save(campaignID, t)
	if err != nil {
		panic(err)
	}

	return nil
}

func failed(campaignID, targetID string) error {
	campaignsm.RLock()
	c, ok := campaigns[campaignID]
	campaignsm.RUnlock()
	if !ok {
		return ErrNotFound
	}

	c.m.Lock()
	defer c.m.Unlock()

	t, ok := c.originating[targetID]
	if !ok {
		return ErrNotFound
	}

	delete(c.originating, targetID)
	if t.Attempts >= c.c.GetMaxAttempts() {
		go emit(EventFail, *t)
		db.delete(campaignID, t.GetId())
		return nil
	}

	t.NextAttemptTime = t.LastAttemptTime + c.c.NextAttemptDelay
	c.l.PushFront(t)
	err := db.save(campaignID, t)
	if err != nil {
		panic(err)
	}

	return nil
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
			c:              c,
			l:              list.New(),
			originating:    map[string]*Target{},
			failedIn:       time.Minute,
			waitForConnect: time.Hour,
			m:              &sync.Mutex{},
		}

		go cmp.targetsCleaner()
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

	delete(c.originating, targetID)

	for e := c.l.Front(); e != nil; e = e.Next() {
		t := e.Value.(*Target)
		if t.GetId() == targetID {
			c.l.Remove(e)
			db.delete(campaignID, targetID)
			return
		}
	}
}

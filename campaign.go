package stewdy

import (
	"container/list"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

var (
	campaigns  = map[string]*campaign{}
	campaignsm sync.RWMutex
)

var (
	ErrNoQID          = errors.New("no queue id provided")
	ErrEmptyTimeTable = errors.New("timetable is empty")
)

type campaign struct {
	c                Campaign
	l                *list.List
	e                *list.Element
	nextAttemptTime  int64
	currentBatchSize int32
	originating      map[string]*Target
	answered         map[string]*Target
	connected        map[string]*Target
	waitForAnswer    time.Duration
	waitForConnect   time.Duration
	waitForHangup    time.Duration
	m                *sync.Mutex
}

func (c *campaign) addTargets(targets []*Target) {
	c.m.Lock()
	for _, v := range targets {
		c.l.PushBack(v)
	}

	c.m.Unlock()
}

func (c *campaign) next(n int32) []*Target {
	return c.nextAtTime(n, time.Now())
}

func (c *campaign) nextAtTime(n int32, t time.Time) []*Target {
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

	eToRemove := []*list.Element{}
	for e := c.e; e != nil && len(res) < int(n); e = e.Next() {
		t := e.Value.(*Target)
		if t.GetAttempts() >= c.c.GetMaxAttempts() {
			defer func() {
				c.l.Remove(e)
				go emit(EventFail, *t)
			}()
			continue
		}

		if nat := t.GetNextAttemptTime(); nat > attemptTime {
			c.nextAttemptTime = nat
			// elements are sorted by NextAttempt time, so no need to find next target
			break
		}

		t.Attempts++
		t.LastAttemptTime = time.Now().Unix()
		c.originating[t.GetId()] = t
		res = append(res, t)
		eToRemove = append(eToRemove, e)
	}

	if len(res) == 0 {
		return res
	}

	err := db.saveManyTargets(c.c.GetId(), res)
	if err != nil {
		panic(err)
	}

	defer func() {
		for _, e := range eToRemove {
			c.l.Remove(e)
		}
	}()

	return res
}

func (c *campaign) freeSlots() int32 {
	free, queued := queueStat(c.c.GetQueueID())
	qFreeSlots := int32(free - queued)
	if qFreeSlots < 0 {
		return 0
	}

	c.m.Lock()
	defer c.m.Unlock()

	currentCalls := int32(len(c.originating) + len(c.answered) + len(c.connected))
	freeSlots := c.c.ConcurrentCalls - currentCalls
	if freeSlots <= 0 {
		c.calcBatchSize(0)
		return 0
	}

	if c.c.Intensity > 0 && freeSlots > c.c.Intensity {
		freeSlots = c.c.Intensity
	}

	if freeSlots > qFreeSlots {
		freeSlots = qFreeSlots
	}

	return c.calcBatchSize(freeSlots)
}

func (c *campaign) calcBatchSize(max int32) int32 {
	if c.currentBatchSize == 0 {
		c.currentBatchSize = 2
	} else {
		c.currentBatchSize = c.currentBatchSize * 2
	}

	if max < c.currentBatchSize {
		c.currentBatchSize = max
	}

	if c.c.BatchSize > 0 && c.currentBatchSize > c.c.BatchSize {
		c.currentBatchSize = c.c.BatchSize
	}

	return c.currentBatchSize
}

func (c *campaign) originator() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		c.originateNext()
	}
}

func (c *campaign) originateNext() {
	freeSlots := c.freeSlots()
	if freeSlots <= 0 {
		return
	}

	targets := c.next(freeSlots)
	if len(targets) == 0 {
		return
	}

	for _, t := range targets {
		emit(EventOriginate, *t)
	}
}

// sort does not lock mutex so you should do it by yourself.
func (c *campaign) sort() {
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

func (c *campaign) targetsCleaner() {
	ticker := time.NewTicker(c.waitForAnswer)
	for now := range ticker.C {
		c.cleanTargets(now)
	}
}

func (c *campaign) cleanTargets(now time.Time) {
	c.m.Lock()
	defer c.m.Unlock()

	failedTime := now.Add(-1 * c.waitForAnswer).Unix()
	notConnectedTime := now.Add(-1 * c.waitForConnect).Unix()
	notHangupedTime := now.Add(-1 * c.waitForHangup).Unix()
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

	for k, v := range c.connected {
		if v.ConnectTime <= notHangupedTime {
			delete(c.connected, k)
			go emit(EventHangup, *v)
		}
	}

	if len(updated) > 0 {
		err := db.saveManyTargets(c.c.GetId(), updated)
		if err != nil {
			panic(err)
		}
	}

	if len(deleted) > 0 {
		db.deleteMany(c.c.GetId(), deleted)
	}
}

func answered(targetID, uniqueID string) error {
	c, ok := findCampaignByTargetID(targetID)
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

	err := db.save(c.c.GetId(), t)
	if err != nil {
		panic(err)
	}

	return nil
}

func connected(uniqueID, operatorID string) error {
	c, ok := findCampaignByUniqueID(uniqueID)
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
	c.connected[uniqueID] = t
	err := db.save(c.c.GetId(), t)
	if err != nil {
		panic(err)
	}

	return nil
}

func failed(targetID string) error {
	c, ok := findCampaignByTargetID(targetID)
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
		db.delete(c.c.GetId(), t.GetId())
		return nil
	}

	t.NextAttemptTime = t.LastAttemptTime + c.c.NextAttemptDelay
	c.l.PushFront(t)
	err := db.save(c.c.GetId(), t)
	if err != nil {
		panic(err)
	}

	return nil
}

func findCampaignByTargetID(targetID string) (*campaign, bool) {
	campaignsm.RLock()
	defer campaignsm.RUnlock()

	for _, c := range campaigns {
		if _, ok := c.originating[targetID]; ok {
			return c, true
		}

		if _, ok := c.connected[targetID]; ok {
			return c, true
		}
	}

	return nil, false
}

func findCampaignByUniqueID(uniqueID string) (*campaign, bool) {
	campaignsm.RLock()
	defer campaignsm.RUnlock()

	for _, c := range campaigns {
		if _, ok := c.answered[uniqueID]; ok {
			return c, true
		}
	}

	return nil, false
}

func hanguped(uniqueID string) error {
	c, ok := findCampaignByUniqueID(uniqueID)
	if !ok {
		return ErrNotFound
	}

	c.m.Lock()
	defer c.m.Unlock()

	t, ok := c.answered[uniqueID]
	if ok {
		if t.Attempts >= c.c.GetMaxAttempts() {
			go func() {
				emit(EventHangup, *t)
				emit(EventFail, *t)
			}()
			db.delete(c.c.GetId(), t.GetId())

			return nil
		}

		t.NextAttemptTime = t.LastAttemptTime + c.c.NextAttemptDelay
		c.l.PushFront(t)
		err := db.save(c.c.GetId(), t)
		if err != nil {
			panic(err)
		}

		go emit(EventHangup, *t)

		return nil
	}

	t, ok = c.connected[uniqueID]
	if !ok {
		return ErrNotFound
	}

	go emit(EventHangup, *t)

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
		cmp = &campaign{
			c:              c,
			l:              list.New(),
			originating:    map[string]*Target{},
			waitForAnswer:  time.Minute,
			waitForConnect: time.Hour,
			waitForHangup:  time.Hour,
			m:              &sync.Mutex{},
		}

		if c.WaitForAnswer > 0 {
			cmp.waitForAnswer = time.Duration(c.WaitForAnswer) * time.Second
		}
		if c.WaitForConnect > 0 {
			cmp.waitForConnect = time.Duration(c.WaitForConnect) * time.Second
		}
		if c.MaxCallDuration > 0 {
			cmp.waitForHangup = time.Duration(c.MaxCallDuration) * time.Second
		}

		campaigns[c.GetId()] = cmp
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

	err := db.saveManyTargets(campaignID, targets)
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

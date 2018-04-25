package stewdy

import (
	"errors"
	"os"
	"os/signal"

	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

var db *database

// DB Errors
var (
	ErrNotFound        = errors.New("not found")
	ErrNotProtoMessage = errors.New("item is not a proto.Message")
	ErrNoIdProvided    = errors.New("no id provided")
	ErrShouldBeSlice   = errors.New("data should be slice of inteface{}")
)

const campaignsBucket = "campaign"

type database struct {
	db *bolt.DB
}

type idGetter interface {
	GetId() string
}

func Init(fileName string) {
	if len(fileName) == 0 {
		fileName = "./stewdy.db"
	}

	boltDB, err := bolt.Open(fileName, 0644, nil)
	if err != nil {
		panic(err)
	}

	db = &database{boltDB}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			db.db.Close()
			close(signalChan)
			os.Exit(0)
		}
	}()
}

func (d *database) getCampaign(id string) (Campaign, error) {
	var campaign Campaign
	encoded, err := d.get(campaignsBucket, id)
	if err != nil {
		return campaign, err
	}

	err = proto.Unmarshal(encoded, &campaign)

	return campaign, err
}

func (d *database) saveCampaign(c Campaign) error {
	return d.save(campaignsBucket, &c)
}

func (d *database) getTarget(cID, tID string) (Target, error) {
	var target Target
	encoded, err := d.get(cID, tID)
	if err != nil {
		return target, err
	}

	err = proto.Unmarshal(encoded, &target)

	return target, err
}

func (d *database) saveTarget(cID string, t Target) error {
	return d.save(cID, &t)
}

func (d *database) delete(b, k string) {
	err := d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b))
		if b == nil {
			return nil
		}

		return b.Delete([]byte(k))
	})

	if err != nil {
		// errcheck stub, because no errors can be returned from bolt
	}
}

func (d *database) deleteBucket(key string) {
	err := d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(key))
		if b == nil {
			return nil
		}

		return tx.DeleteBucket([]byte(key))
	})

	if err != nil {
		// errcheck stub, because no errors can be returned from bolt
	}
}

func (d *database) deleteMany(b string, keys []string) {
	err := d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b))
		if b == nil {
			return nil
		}

		for _, k := range keys {
			err := b.Delete([]byte(k))
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		// errcheck stub, because no errors can be returned from bolt
	}
}

func (d *database) get(b, k string) ([]byte, error) {
	var content []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(b))
		if b == nil {
			return ErrNotFound
		}

		content = b.Get([]byte(k))
		if content == nil {
			return ErrNotFound
		}

		return nil
	})

	return content, err
}

func (d *database) save(b string, item proto.Message) error {
	err := d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(b))
		if err != nil {
			return err
		}

		id, err := getId(item)
		if err != nil {
			return err
		}

		encoded, err := proto.Marshal(item)
		if err != nil {
			return err
		}

		return b.Put([]byte(id), encoded)
	})

	return err
}

func (d *database) saveManyTargets(b string, items []*Target) error {
	if len(items) == 0 {
		return nil
	}
	err := d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(b))
		if err != nil {
			return err
		}

		for _, item := range items {
			encoded, err := proto.Marshal(item)
			if err != nil {
				return err
			}

			err = b.Put([]byte(item.GetId()), encoded)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (d *database) saveMany(b string, items []interface{}) error {
	if len(items) == 0 {
		return nil
	}

	err := d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(b))
		if err != nil {
			return err
		}

		for _, v := range items {
			item, ok := v.(proto.Message)
			if !ok {
				return ErrNotProtoMessage
			}

			id, err := getId(item)
			if err != nil {
				return err
			}

			encoded, err := proto.Marshal(item)
			if err != nil {
				return err
			}

			err = b.Put([]byte(id), encoded)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func getId(i interface{}) (string, error) {
	t, ok := i.(idGetter)
	if !ok {
		return "", ErrNoIdProvided
	}

	return t.GetId(), nil
}

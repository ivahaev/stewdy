package stewdy

import (
	"log"
	"testing"
)

func TestDB(t *testing.T) {
	testBucket := "test"
	db.deleteBucket(testBucket)
	defer db.deleteBucket(testBucket)

	t.Run("Targets", func(t *testing.T) {
		tr := Target{Id: "T1", CampaignID: "C1"}
		err := db.saveTarget(testBucket, tr)
		if err != nil {
			t.Fatal(err)
		}

		trg, err := db.getTarget(testBucket, tr.GetId())
		if err != nil {
			t.Fatal(err)
		}

		if trg.CampaignID != tr.CampaignID {
			t.Fatalf("trg.CampaignID = %s, expected %s", trg.CampaignID, tr.CampaignID)
		}
	})

	t.Run("Campaigns", func(t *testing.T) {
		c := Campaign{Id: "11", QueueID: "11"}
		err := db.saveCampaign(c)
		if err != nil {
			t.Fatal(err)
		}

		cmp, err := db.getCampaign(c.GetId())
		if err != nil {
			t.Fatal(err)
		}

		if cmp.QueueID != c.QueueID {
			t.Errorf("cmp.QueueID = %s, expected %s", cmp.QueueID, c.QueueID)
		}

		_, err = db.getCampaign("123321")
		if err == nil {
			log.Fatal("db.getCampaign(\"123321\") returns no error")
		}
	})

	t.Run("Many", func(t *testing.T) {
		campaigns := []interface{}{
			&Campaign{
				Id:      "C1",
				QueueID: "Q11",
			},
			&Campaign{
				Id:      "C2",
				QueueID: "Q22",
			},
		}

		err := db.saveMany(testBucket, campaigns)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ManyTargets", func(t *testing.T) {
		targets := []*Target{
			{
				Id:          "1",
				PhoneNumber: "11",
			},
			{
				Id:          "2",
				PhoneNumber: "22",
			},
		}

		err := db.saveManyTargets(testBucket, targets)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("DeleteMany", func(t *testing.T) {
		targets := []*Target{
			{
				Id:          "1",
				PhoneNumber: "11",
			},
			{
				Id:          "2",
				PhoneNumber: "22",
			},
		}

		db.saveManyTargets(testBucket, targets)
		ids := []string{"1", "2"}
		for _, id := range ids {
			trg, err := db.getTarget(testBucket, id)
			if err != nil {
				t.Fatal(err)
			}
			if trg.GetId() != id {
				t.Fatalf("trg.Id = %s, expected %s", trg.GetId(), id)
			}
		}

		db.deleteMany(testBucket, ids)

		for _, id := range ids {
			_, err := db.getTarget(testBucket, id)
			if err == nil {
				t.Errorf("no error returned, expected %v", ErrNotFound)
			}
		}
	})
}

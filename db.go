package irl

import (
	"encoding/binary"
	"github.com/boltdb/bolt"
	"math"
)

func AddRun(db *bolt.DB, team string, result *Result) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(team))
		if err != nil {
			return err
		}
		for k, v := range result.Topics["all"] {
			var buff [8]byte
			binary.BigEndian.PutUint64(buff[:], math.Float64bits(v))
			err := b.Put([]byte(k), buff[:])
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func GetRun(db *bolt.DB, team string) (*Result, error) {
	result := new(Result)
	result.RunId = team
	result.Topics = make(map[string]map[string]float64)
	result.Topics["all"] = make(map[string]float64)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(team))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			result.Topics["all"][string(k)] = math.Float64frombits(binary.BigEndian.Uint64(v))
			return nil
		})
	})
	return result, err
}

func GetRuns(db *bolt.DB, teams ...string) (map[string]*Result, error) {
	results := make(map[string]*Result)
	for _, team := range teams {
		var err error
		results[team], err = GetRun(db, team)
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

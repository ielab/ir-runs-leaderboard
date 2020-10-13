package irl

import (
	"encoding/binary"
	"github.com/boltdb/bolt"
	"math"
)

func AddRun(db *bolt.DB, team, run string, result *Result) error {
	return db.Update(func(tx *bolt.Tx) error {
		tb, err := tx.CreateBucketIfNotExists([]byte(team))
		if err != nil {
			return err
		}
		rb, err := tb.CreateBucketIfNotExists([]byte(run))
		if err != nil {
			return err
		}
		for k, v := range result.Topics["all"] {
			var buff [8]byte
			binary.BigEndian.PutUint64(buff[:], math.Float64bits(v))
			err := rb.Put([]byte(k), buff[:])
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func GetRunsForTeam(db *bolt.DB, team string) (map[string]*Result, error) {
	results := make(map[string]*Result)
	err := db.View(func(tx *bolt.Tx) error {

		tb := tx.Bucket([]byte(team))
		if tb == nil {
			return nil
		}
		return tb.ForEach(func(run, v []byte) error {

			result := new(Result)
			result.RunId = string(run)
			result.Topics = make(map[string]map[string]float64)
			result.Topics["all"] = make(map[string]float64)
			// https://github.com/boltdb/bolt/issues/295#issuecomment-72476443
			tr := tb.Bucket(run)
			if tr == nil {
				return nil
			}
			err := tr.ForEach(func(k, v []byte) error {
				result.Topics["all"][string(k)] = math.Float64frombits(binary.BigEndian.Uint64(v))
				return nil
			})

			results[string(run)] = result
			return err
		})
	})

	return results, err
}

package irl

import (
	"fmt"
	"testing"
)

func TestEval(t *testing.T) {
	result, err := Eval("trec_eval/trec_eval", "-q", "test_data/task1.test.abs.qrels", "test_data/sheffield-bm25.res")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	fmt.Println(result.RunId)
	fmt.Println(result.Topics["all"]["num_ret"])
}

package irl

import (
	"bufio"
	"github.com/go-errors/errors"
	"io"
	"strconv"
	"strings"
)

// Result represents an entire output from trec_eval.
type Result struct {
	RunId  string                        // This is a special line in the output that can be a string.
	Topics map[string]map[string]float64 // Each topic is keyed to the list of measures specified.
}

// Decode will decode a reader into a Result struct.
func Decode(r io.Reader) (*Result, error) {

	var result Result
	result.Topics = make(map[string]map[string]float64)

	// Read the input line by line.
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {

		line := scanner.Text()
		columns := strings.Fields(line)
		if len(columns) != 3 {
			return nil, errors.New("invalid number of columns in evaluation output")
		}

		name := columns[0]
		topic := columns[1]

		// Special case: the runid must be parsed separately.
		if topic == "all" && name == "runid" {
			result.RunId = columns[2]
			continue
		}

		v, err := strconv.ParseFloat(columns[2], 64)
		if err != nil {
			return nil, err
		}

		// Make a new map if the measure has not been seen for this topic.
		if _, ok := result.Topics[topic]; !ok {
			result.Topics[topic] = make(map[string]float64)
		}

		// Add the parsed value to this measure for this topic.
		result.Topics[topic][name] = v
	}

	return &result, nil
}

func ExtractRunIdFromRun(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	columns := strings.Fields(scanner.Text())
	return columns[5], nil
}

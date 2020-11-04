package irl

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Eval calls the trec_eval binary at bin with the arguments, the qrels file, and run file
// and returns a set of results.
func Eval(bin, args, qrels, run string) (*Result, error) {
	cmd := exec.Command(bin, append(append(strings.Fields(args), qrels), run)...)

	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	fmt.Println(cmd.Args)

	_ = cmd.Start()

	s := bufio.NewScanner(bufio.NewReader(r))
	// Skip the first line.
	s.Scan()
	var buff bytes.Buffer
	for s.Scan() {
		// Note that when using this method of reading from stdout,
		// it does not add the newlines. Therefore, they need to be
		// added back in, see below.
		_, err = buff.Write(append(s.Bytes(), '\n'))
		if err != nil {
			return nil, err
		}
	}

	// This is the decode method that is implemented in this package.
	result, err := Decode(&buff)
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

# IR Runs Leaderboard (irl)

This package provides a server application that can support different teams uploading standard trec_eval style runs for a leaderboard competition.
The server can be configured with different evaluation measures, and a sorting on one evaluation measure can be supplied.
Teams can be provided a unique URL provided by organisers to upload their runs.
At the moment, only one run per team is available.

Other features:

 - Every run submitted is saved to the disk and timestamped for historical purposes.
 - Runs are evaluated using the trec_eval binary, and all evaluation measures supported by trec_eval can be columns in the leaderboard.

## Installation:

Requirements:

 - `go`
 - `make`

```bash
$ git clone --recurse-submodules git@github.com:ielab/ir-runs-leaderboard.git
$ cd trec_eval
$ make
$ cd ../
$ go build -o cmd/irl/*
```

## Running:

First, create a `config.json` file based on the existing `sample.config.json`.
The configuration items of this file are:

 - database: the file that persists the most recent evaluation results of a team.
 - measures: a list of trec_eval measures to include as columns in the leaderboard.
 - sort_on: the measure to use as the sorting criteria in the leaderboard.
 - trec_eval:
   - bin: the path to the trec_eval binary.
   - args: additional arguments to the trec_eval binary.
   - qrels: the path to the qrels file used for evaluation.
 - teams: key-value pairs, where the keys are the team names and the values are the secrets that teams use to access their file upload page.  

```bash
./irl
```

This will start a server on localhost, post 8088. Note that the local assets are not bundled into the server.
That means you need to run the server in this directory. You can skip the build step if modifying the server by just executing `go run cmd/irl/main.go`.

## Upload URLs:

Teams can access their private upload page at the following url:

```bash
localhost:8088/upload?secret=SUPERSECRET
```

Where `SUPERSECRET` is the value specified for that team in the `config.json` file as documented above.
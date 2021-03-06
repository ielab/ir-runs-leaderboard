package main

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
	"github.com/ielab/irl"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type server struct {
	db      *bolt.DB
	config  config
	secrets map[string]string
}

type config struct {
	Teams    map[string]string `json:"teams"`
	Database string            `json:"database"`
	Measures []string          `json:"measures"`
	SortOn   string            `json:"sort_on"`
	TrecEval struct {
		Bin   string `json:"bin"`
		Args  string `json:"args"`
		Qrels string `json:"qrels"`
	} `json:"trec_eval"`
	Title  string `json:"title"`
	Header string `json:"header"`
}

type result struct {
	Team     string
	Measures []float64
}

type response struct {
	Measures []string
	Results  []result
}

type ErrorPage struct {
	Error    string
	BackLink string
}

type Page struct {
	Content string
}

func main() {

	var s server

	// Handle the config.
	f, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	err = json.NewDecoder(f).Decode(&s.config)
	if err != nil {
		panic(err)
	}

	// Handle the scores database.
	s.db, err = bolt.Open(s.config.Database, 0777, bolt.DefaultOptions)
	if err != nil {
		panic(err)
	}

	// Create a reverse mapping of secret->team
	s.secrets = make(map[string]string)
	for k, v := range s.config.Teams {
		s.secrets[v] = k
	}

	r := gin.Default()
	r.LoadHTMLFiles("assets/index.html", "assets/upload.html", "assets/error.html")
	r.GET("/", s.index)
	r.GET("/upload", s.addRunView)
	r.POST("/upload", s.addRun)
	r.Static("/static/", "./assets/static")
	panic(r.Run(":8088"))

}

func (s server) index(c *gin.Context) {
	r, err := s.buildTeamsTable()
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", ErrorPage{Error: "Error Loading Page", BackLink: "/"})
		return
	}
	c.HTML(http.StatusOK, "index.html", struct {
		Title    string
		Header   string
		Measures []string
		Results  []result
	}{
		Title:    s.config.Title,
		Header:   s.config.Header,
		Measures: r.Measures,
		Results:  r.Results,
	})
	return
}

func (s server) addRunView(c *gin.Context) {
	secret := c.Query("secret")
	if team, ok := s.secrets[secret]; ok {
		c.HTML(http.StatusOK, "upload.html", struct {
			Secret string
			Team   string
			Title  string
		}{
			Secret: secret,
			Team:   team,
			Title:  s.config.Title,
		})
		return
	}
	c.HTML(http.StatusUnauthorized, "error.html", ErrorPage{Error: "Invalid Secret", BackLink: "/"})
	return
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (s server) addRun(c *gin.Context) {
	secret := c.PostForm("secret")
	team := c.PostForm("team")

	// First, check to see if the secret is for this team.
	if s.secrets[secret] != team {
		c.HTML(http.StatusUnauthorized, "error.html", ErrorPage{Error: "Invalid Secret", BackLink: "/"})
		return
	}

	// Grab the run file from the POST form.
	file, err := c.FormFile("run")
	if err != nil {
		c.HTML(http.StatusUnauthorized, "error.html", ErrorPage{Error: "No Run File Uploaded", BackLink: "/"})
		return
	}

	permittedFormat := []string{"TXT", "RES", "res", "txt"}
	filenameParts := strings.Split(file.Filename, ".")
	if !contains(permittedFormat, filenameParts[len(filenameParts)-1]) {
		c.HTML(http.StatusBadRequest, "error.html", ErrorPage{Error: "Wrong File Format", BackLink: "/"})
		return
	}

	fpath := strings.Replace(path.Join("runs", team), " ", "-", -1)
	err = os.MkdirAll(fpath, 0777)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "error.html", ErrorPage{Error: "Team Directory Creation Error", BackLink: "/"})
		return
	}

	rf, err := file.Open()
	if err != nil {
		panic(err)
	}
	runId, err := irl.ExtractRunIdFromRun(rf)

	fname := fmt.Sprintf("%s.%s.%d.run", team, runId, time.Now().Unix())
	runPath := strings.Replace(path.Join(fpath, fname), " ", "-", -1)
	err = c.SaveUploadedFile(file, runPath)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "error.html", ErrorPage{Error: "File Saving Error", BackLink: "/"})
		return
	}

	// Open the file and evaluate it from the disk.
	result, err := irl.Eval(s.config.TrecEval.Bin, s.config.TrecEval.Args, s.config.TrecEval.Qrels, runPath)
	if err != nil {
		fmt.Println(err)
		c.HTML(http.StatusUnauthorized, "error.html", ErrorPage{Error: "Invalid Run File", BackLink: "/"})
		return
	}

	// Update the run for this team.
	err = irl.AddRun(s.db, team, result.RunId, result)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "error.html", ErrorPage{Error: "Result Update Error", BackLink: "/"})
		return
	}

	c.Redirect(http.StatusFound, "/")
	return
}

func (s server) buildTeamsTable() (*response, error) {
	var r response
	r.Measures = s.config.Measures

	var sortOn int
	for i := 0; i < len(r.Measures); i++ {
		if r.Measures[i] == s.config.SortOn {
			sortOn = i
		}
	}

	for team := range s.config.Teams {
		runs, err := irl.GetRunsForTeam(s.db, team)
		if err != nil {
			return nil, err
		}

		if len(runs) == 0 {
			continue
		}

		for name, run := range runs {
			var res result
			res.Team = fmt.Sprintf("%s (%s)", team, name)
			res.Measures = make([]float64, len(r.Measures))
			for i, measure := range r.Measures {
				res.Measures[i] = run.Topics["all"][measure]
			}
			r.Results = append(r.Results, res)
		}
	}
	
	sort.Slice(r.Results, func(i, j int) bool {
		return r.Results[i].Measures[sortOn] > r.Results[j].Measures[sortOn]
	})

	return &r, nil
}

func getAllRunFilenames() ([]string, error) {
	var filenames []string
	e := filepath.Walk("runs", func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if strings.ContainsAny(info.Name(), ".") {
				runName := strings.Replace(strings.Split(info.Name(), ".")[0], "-", " ", 2)
				filenames = append(filenames, runName)
			}
		}
		return nil
	})

	if e != nil {
		return nil, e
	}

	return filenames, nil
}

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
}

type result struct {
	Team     string
	Measures []float64
}

type response struct {
	Measures []string
	Results  []result
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
	r.LoadHTMLFiles("assets/index.html", "assets/upload.html")
	r.GET("/", s.index)
	r.GET("/upload", s.addRunView)
	r.POST("/upload", s.addRun)
	r.Static("/static/", "./assets/static")
	panic(r.Run(":8088"))

}

func (s server) index(c *gin.Context) {
	r, err := s.buildTeamsTable()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		panic(err)
		return
	}
	c.HTML(http.StatusOK, "index.html", r)
}

func (s server) addRunView(c *gin.Context) {
	secret := c.Query("secret")
	if team, ok := s.secrets[secret]; ok {
		c.HTML(http.StatusOK, "upload.html", struct {
			Secret string
			Team   string
		}{
			Secret: secret,
			Team:   team,
		})
		return
	}
	c.String(http.StatusUnauthorized, "invalid secret")
	return
}

func (s server) addRun(c *gin.Context) {
	secret := c.PostForm("secret")
	team := c.PostForm("team")

	// First, check to see if the secret is for this team.
	if s.secrets[secret] != team {
		c.String(http.StatusUnauthorized, "invalid secret")
		return
	}

	// Grab the run file from the POST form.
	file, err := c.FormFile("run")
	if err != nil {
		c.Status(http.StatusInternalServerError)
		panic(err)
		return
	}

	// Save the file to the disk.
	fpath := strings.Replace(path.Join("runs", team), " ", "-", -1)
	err = os.MkdirAll(fpath, 0777)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		panic(err)
		return
	}
	fname := fmt.Sprintf("%s.%d.run", team, time.Now().Unix())
	runPath := strings.Replace(path.Join(fpath, fname), " ", "-", -1)
	err = c.SaveUploadedFile(file, runPath)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		panic(err)
		return
	}

	// Open the file and evaluate it from the disk.
	result, err := irl.Eval(s.config.TrecEval.Bin, s.config.TrecEval.Args, s.config.TrecEval.Qrels, runPath)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		panic(err)
		return
	}

	// Update the run for this team.
	err = irl.AddRun(s.db, team, result)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		panic(err)
		return
	}

	c.Redirect(http.StatusFound, "/")
	return
}

func (s server) buildTeamsTable() (*response, error) {
	var r response
	r.Measures = s.config.Measures
	r.Results = make([]result, len(s.config.Teams))

	var sortOn int
	for i := 0; i < len(r.Measures); i++ {
		if r.Measures[i] == s.config.SortOn {
			sortOn = i
		}
	}

	var i int
	for k := range s.config.Teams {
		result, err := irl.GetRun(s.db, k)
		if err != nil {
			return nil, err
		}

		r.Results[i].Team = k
		r.Results[i].Measures = make([]float64, len(r.Measures))
		for j, measure := range r.Measures {
			r.Results[i].Measures[j] = result.Topics["all"][measure]
		}
		i++
	}

	sort.Slice(r.Results, func(i, j int) bool {
		return r.Results[i].Measures[sortOn] > r.Results[j].Measures[sortOn]
	})

	return &r, nil
}

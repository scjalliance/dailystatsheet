package dailystatsheet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gentlemanautomaton/stathat"
)

// Request is the inbound request
type Request struct {
	UserAgentString string    `json:"userAgent"`
	HookURL         string    `json:"hook"`
	StatHatToken    string    `json:"token"`
	TimeZone        string    `json:"tz"`
	Stat            []string  `json:"stat"`
	StartDate       time.Time `json:"startDate"`
}

// Run the updater
func Run(w http.ResponseWriter, r *http.Request) {
	var request Request
	j := json.NewDecoder(r.Body)
	err := j.Decode(&request)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "NOK: body parse error")
		return
	}

	// UserAgentString
	stathat.UserAgent = `github.com/scjalliance/dailystatsheet`
	if request.UserAgentString != "" {
		stathat.UserAgent = request.UserAgentString
	}

	// Hook
	webhookURL := request.HookURL
	if webhookURL == "" {
		fmt.Println(err)
		fmt.Fprint(w, "NOK: missing hook")
		return
	}

	// StatHatToken
	token := request.StatHatToken
	if token == "" {
		fmt.Println(err)
		fmt.Fprint(w, "NOK: missing token")
		return
	}
	s := stathat.New().Token(token)

	// TimeZone
	timeZone := `America/Los_Angeles`
	if request.TimeZone != "" {
		timeZone = request.TimeZone
	}
	tz, err := time.LoadLocation(timeZone)

	// Stat
	stats := request.Stat
	if stats == nil || len(stats) < 1 {
		fmt.Println("request.Stat is nil or empty")
		fmt.Fprint(w, "NOK: missing stats.")
		return
	}

	// StartDate
	startDate := request.StartDate
	if startDate.IsZero() {
		startDate = time.Now().In(tz).AddDate(0, 0, -1) // go back one day because we want to get yesterday's stats, not today's
	}

	// // //

	interval := `1h`
	period := `23h`
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "NOK: timezone error")
		return
	}
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, tz)
	statBatchSize := 5

	var datasets []stathat.Dataset

	var statBatches [][]string
	for statBatchSize < len(stats) {
		stats, statBatches = stats[statBatchSize:], append(statBatches, stats[0:statBatchSize:statBatchSize])
	}
	statBatches = append(statBatches, stats)
	for b := 0; b < len(statBatches); b++ {
		data, err := s.Get(stathat.GetOptions{
			Start:    &start,
			Interval: interval,
			Period:   period,
		}, statBatches[b]...)
		if err != nil {
			fmt.Printf("Error occurred: %s\n", err)
			fmt.Fprintf(w, "NOK: stat get batch %d", b)
			return
		}

		datasets = append(datasets, data...)
	}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode([]struct {
		Data []stathat.Dataset
	}{{
		Data: datasets,
	}})

	_, err = http.Post(webhookURL, `application/json`, b)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "NOK: hook error")
		return
	}

	fmt.Fprint(w, "OK")
}

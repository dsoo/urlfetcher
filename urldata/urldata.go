package urldata

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// Response represents data retrieved from an URL.
type Response struct {
	URL       string
	Body      string
	Timestamp time.Time
}

// Job represents an individual job request
type Job struct {
	ID       int64
	URL      string
	Status   string    // Enum of status - waiting, success, error
	Response *Response // The result data for the job
}

// "Global" state for the package representing data and jobs
var jobQueue = make(chan int64, 1000)
var jobs = make(map[int64]*Job)
var responses = make(map[string]*Response)
var curJobID = int64(0)

// AddJob adds a new job to the work queue
func AddJob(url string) Job {
	// FIXME: Make this atomic
	jobID := atomic.AddInt64(&curJobID, 1)
	job := Job{
		ID:       jobID,
		URL:      url,
		Status:   "waiting",
		Response: nil,
	}
	jobs[jobID] = &job

	jobQueue <- job.ID
	return job
}

// GetJob returns the job associated with the ID
func GetJob(id int64) *Job {
	return jobs[id]
}

// GetResponse returns the response data associated with the URL
func GetResponse(url string) *Response {
	return responses[url]
}

func doJob(jobID int64) {
	// Check if we already have data in the cache - if so, we can fill it right away
	// and skip adding it to the work queue.
	// Returns the URL data associated with the URL, returning the cached
	// data.
	// FIXME: Optimize to reduce impact of rapid concurrent requests for the same URL.
	fmt.Println("Fetching job", jobID)
	job := jobs[jobID]

	job.Status = "fetching"
	resp, err := http.Get(job.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	response := Response{
		URL:       job.URL,
		Body:      string(body),
		Timestamp: time.Now(),
	}
	responses[job.URL] = &response
	job.Response = &response
	job.Status = "done"
}

func fetchWorker(jobQueue chan int64) {
	// Continually fetch jobIDs off the channel and
	// fetch/update their URL data.
	fmt.Println("running worker")
	for {
		jobID := <-jobQueue
		doJob(jobID)
	}
}

// RunWorkers runs numWorkers workers that pull jobs off the queue.
func RunWorkers(numWorkers int) {
	// Initialize the job queue channel
	// Instantiate a bunch of works.

	for i := 0; i < numWorkers; i++ {
		go fetchWorker(jobQueue)
	}
}

package urldata

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/graphql-go/graphql"
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

// SchemaConfig configures the graphql schema and callbacks
func SchemaConfig() graphql.SchemaConfig {
	responseType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Response",
		Fields: graphql.Fields{
			"url": &graphql.Field{
				Type:        graphql.String,
				Description: "The URL that was retrieved using HTTP GET",
			},
			"body": &graphql.Field{
				Type:        graphql.String,
				Description: "The body of the HTTP response",
			},
		},
	})

	jobType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Job",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.Int,
				Description: "Unique ID for the job",
			},
			"url": &graphql.Field{
				Type:        graphql.String,
				Description: "An URL to be retrieved via HTTP GET",
			},
			"status": &graphql.Field{
				Type:        graphql.String,
				Description: "Simple status string for the job. Can be waiting, fetching, done, done - cached",
			},
			"response": &graphql.Field{
				Type:        responseType,
				Description: "Response data from the URL to be retrieved. May be cached.",
			},
		},
	})
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"jobs": &graphql.Field{
				Type:        graphql.NewList(jobType),
				Description: "Retrieve information about all jobs on the server",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return GetJobs(), nil
				},
			},
			"job": &graphql.Field{
				Type:        jobType,
				Description: "Retrieve parameters of a job, given the ID of the job",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Description: "id of the job",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, err := strconv.Atoi(p.Args["id"].(string))
					if err != nil {
						return nil, err
					}
					return GetJob(int64(id)), nil
				},
			},
			"responses": &graphql.Field{
				Type:        graphql.NewList(responseType),
				Description: "Retrieve information about all responses on the server",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return GetResponses(), nil
				},
			},
			"response": &graphql.Field{
				Type:        responseType,
				Description: "Retrieve response data for a particular URL.",
				Args: graphql.FieldConfigArgument{
					"url": &graphql.ArgumentConfig{
						Description: "url that we requested",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					url := p.Args["url"].(string)
					return GetResponse(url), nil
				},
			},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"addJob": &graphql.Field{
				Type:        jobType,
				Description: "Add a new urlfetch job to the queue.",
				Args: graphql.FieldConfigArgument{
					"url": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					job := AddJob(params.Args["url"].(string))
					return job, nil
				},
			},
		},
	})

	schemaConfig := graphql.SchemaConfig{Query: rootQuery,
		Mutation: rootMutation}

	return schemaConfig
}

// "Global" state for the package representing data and jobs
var jobQueue = make(chan int64, 1000)
var jobs = make(map[int64]*Job)
var responses = make(map[string]*Response)
var curJobID = int64(0)

// AddJob adds a new job to the work queue
func AddJob(url string) Job {
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

// GetJobs returns all jobs stored by this server as a slice
func GetJobs() []*Job {
	sliceJobs := []*Job{}
	for _, job := range jobs {
		sliceJobs = append(sliceJobs, job)
	}
	return sliceJobs
}

// GetResponse returns the response data associated with the URL
func GetResponse(url string) *Response {
	return responses[url]
}

// GetResponses returns all responses stored by this server as a slice
func GetResponses() []*Response {
	sliceResponses := []*Response{}
	for _, response := range responses {
		sliceResponses = append(sliceResponses, response)
	}
	return sliceResponses
}

func doJob(jobID int64) {
	// Check if we already have data in the cache - if so, we can fill it right away
	// and skip adding it to the work queue.
	// Returns the URL data associated with the URL, returning the cached
	// data.
	// FIXME: Optimize to reduce impact of rapid concurrent requests for the same URL.
	fmt.Println("Fetching job", jobID)
	job := jobs[jobID]

	// Check the cache
	response, ok := responses[job.URL]
	if ok && (time.Now().Sub(response.Timestamp).Hours() < 1.0) {
		// Immediately fill with cache and finish the job.
		job.Response = response
		job.Status = "done - cached"
	} else {
		job.Status = "fetching"
		resp, err := http.Get(job.URL)
		if err != nil {
			job.Response = nil
			job.Status = "error - error with GET"
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			job.Response = nil
			job.Status = "error - error reading body"
		}
		response := &Response{
			URL:       job.URL,
			Body:      string(body),
			Timestamp: time.Now(),
		}
		responses[job.URL] = response
		job.Response = response
		job.Status = "done"
	}
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

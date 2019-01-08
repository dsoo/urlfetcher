package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dsoo/urlfetcher/urldata"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

// configure the graphql schema and callbacks
func schemaConfig() graphql.SchemaConfig {
	responseType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Response",
		Fields: graphql.Fields{
			"url": &graphql.Field{
				Type: graphql.String,
			},
			"body": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

	jobType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Job",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"url": &graphql.Field{
				Type: graphql.String,
			},
			"status": &graphql.Field{
				Type: graphql.String,
			},
			"response": &graphql.Field{
				Type: responseType,
			},
		},
	})
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"job": &graphql.Field{
				Type: jobType,
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
					return urldata.GetJob(int64(id)), nil
				},
			},
			"response": &graphql.Field{
				Type: responseType,
				Args: graphql.FieldConfigArgument{
					"url": &graphql.ArgumentConfig{
						Description: "url that we requested",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					url := p.Args["url"].(string)
					return urldata.GetResponse(url), nil
				},
			},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"addJob": &graphql.Field{
				Type: jobType,
				Args: graphql.FieldConfigArgument{
					"url": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					job := urldata.AddJob(params.Args["url"].(string))
					return job, nil
				},
			},
		},
	})

	schemaConfig := graphql.SchemaConfig{Query: rootQuery,
		Mutation: rootMutation}

	return schemaConfig
}

func main() {
	fmt.Println("running workers")
	urldata.RunWorkers(2)
	fmt.Println("adding jobs")
	urldata.AddJob("https://google.com")
	urldata.AddJob("https://arstechnica.com")

	schema, err := graphql.NewSchema(schemaConfig())
	if err != nil {
		log.Fatalf("failed to create new schema, error: %v", err)
	}

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	http.Handle("/graphql", h)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

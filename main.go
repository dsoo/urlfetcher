package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dsoo/urlfetcher/urldata"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

func main() {
	fmt.Println("running workers")
	urldata.RunWorkers(2)
	fmt.Println("adding jobs")
	urldata.AddJob("https://google.com")
	urldata.AddJob("https://arstechnica.com")

	schema, err := graphql.NewSchema(urldata.SchemaConfig())
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

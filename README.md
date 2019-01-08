## About

This is a super-basic implementation of **urlfetch**, which implements the following:
* A GraphQL service which provides an API to create jobs to fetch data from URLs, as well as
query the status of the job and the HTTP responses.
* If the URL has been fetched in the last hour, it will respond with cached data from a prior request.

Things to note:
* Error handling is at the moment quite questionable. It definitely could be better.
* Related, there are currently no tests.
* There are no mechanisms for cache invalidation or flushing the cache. It will eventually use
up all memory on the system.

## Running
You need a recent version of **go** in order to run this application. Install it
according to documentation. On Macs, it is easiest to install it if you have **brew**
installed by running *brew install go*

Once go is installed, checkout this repository, and you will be able to run this application by running

    go run main.go

This will start up a server at [http://localhost:8080/graphql](http://localhost:8080/graphql)

You can interact with it via the **GraphiQL** browser at that URL.

The app provides a GraphQL Schema which can be browsed using GraphiQL
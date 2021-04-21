# Best Practices in writing Go RESTful API client

Here is a collection of best practices and example fo client library. I put this together while preparing for the interviews. It is easy to forget about some key patterns, which indicates the quality of Golang engineer




- client structure for your client with common parameters like baseURL, http.Client
```go
type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client
	UserAgent string

	// Rate contains the current rate limit for the client as determined by the most recent
	// API call. It is not thread-safe. Please consider using GetRate() instead.
	Rate    Rate
	ratemtx sync.Mutex

	// Services used for communicating with the API
	Account           AccountService
 
 // Optional extra HTTP headers to set on every request to the API.
	headers map[string]string
```

- create service interface for each api section
```go
type Client struct {
 // ....
 
	// Services used for communicating with the API
	Account           AccountService
}
```

- 

- create a new file for each service interface



- use context in every endpoint function

 

## Error handling



# Interview

? example of api for the interview

- create a new project and a folder for your restful api client
- create 


# Examples of well written api clinet library
- https://github.com/digitalocean/godo
- https://github.com/google/go-github/tree/master/github




# References

https://youtu.be/evorkFq3Y5k
https://github.com/digitalocean/godo
https://github.com/google/go-github/tree/master/github
https://www.reddit.com/r/golang/comments/4vfj9e/best_practices_for_writing_a_library_around_a/?utm_source=amp&utm_medium=&utm_content=post_body
https://www.reddit.com/r/golang/comments/dsxvrk/best_practices_for_building_a_restapi/
https://medium.com/@cep21/go-client-library-best-practices-83d877d604ca
https://github.com/tjarratt/go-best-practices

https://cloud.google.com/apis/design/
https://github.com/dhax/go-base
https://www.reddit.com/r/golang/comments/dsxvrk/best_practices_for_building_a_restapi/
Go-swagger generator
https://github.com/go-chi/chi/tree/master/_examples/rest
https://www.reddit.com/r/golang/comments/c5bz48/what_are_some_common_mistakes_when_writing_go/
https://deliveroo.engineering/2019/05/17/testing-go-services-using-interfaces.html
https://github.com/beeker1121/gotodo
Server:
https://github.com/ardanlabs/service
https://itnext.io/structuring-a-production-grade-rest-api-in-golang-c0229b3feedc
https://peter.bourgon.org/go-in-production/
https://opensource.zalando.com/restful-api-guidelines/


Api design
https://stackoverflow.blog/2020/03/02/best-practices-for-rest-api-design/
https://opensource.zalando.com/restful-api-guidelines/
https://yourbasic.org/algorithms/your-basic-api/


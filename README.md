# Best Practices in writing Go RESTful API client

Here is a collection of best practices and example fo client library. I put this together while preparing for the interviews. It is easy to forget about some key patterns, which indicates the quality of Golang engineer


## Overall structure of API client

Client structure contains reusable components like http.Client and configuration. Api endpoints are packaged into Service structure, which is a member of a Client.

Service consists of interface with endpoint functions and Service structure which implements this interface. 

Request are process the following way. There is a client level function to create request, which is passed to http.Client. This top level request generator takes care of common headers like authentication, Accept header, User-agent header, etc. It is also responsible for adding baseURL to endpoint path. 

Request is executed by Do function, which is at the Client level. It takes Request and interface for marshalled response body. 
Before it calls http.Clinet Do function, it checks rate limits from previous response. If limits are binding, it return fake response with 429 status error. Rate limiter values are stored in a Client structure and accessed with mutex lock. Response values from http.Client.Do goes through the following process:
- create new response with extra stuff (rate limits, headers, etc)
- save rate in client
- check for errors
- if there is no error, decode body to provided output interface


Inital response processing is taken care of at Client level. http.Response is wrapped by your own Response structure. You can keep there extra stuff like paginating variables (offset, limit, etc) and limiter rates in the response (parsed http headers  RateLimit-Limit, RateLimit-Remaining, RateLimit-Reset). 
NewResponse function takes http.Client response as argument, parses response headers and populate this extra stuff in the response wrapper structure.

Error check
First define ErrorResponse structure (make sure to define Error() to satisfy Go error interface), which contains reference to http.Response. Then you can create Errors for some others http errors, which you want to treat differently in your application, e.g. rate limits (e.g. RateLimitError), forbidden (AuthError), etc

- Http status >= 200 and < 299 generate no error
- 202 Accepted Response is treated separetely; 202 is when request was accepted but not yet processed. AcceptedError is to treat is separetely from other successful calls
- if http status >=300, then
	- read and unmarshal response body if read body != nil, in case API has message in the response
	- filter through status codes for which you want to generate custom Error
	- for any other error, generate default ResponseError which refers to http.Request





# Details
- start with creating module
```bash
go mod init client
```

- create client main file, client.go

-define constants like baseURL

const (
	libraryVersion = "1.60.0"
	defaultBaseURL = "https://api.digitalocean.com/"
	userAgent      = "godo/" + libraryVersion
	mediaType      = "application/json"

	headerRateLimit     = "RateLimit-Limit"
	headerRateRemaining = "RateLimit-Remaining"
	headerRateReset     = "RateLimit-Reset"
)


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
	Tag           TagsService
 
 // Optional extra HTTP headers to set on every request to the API.
	headers map[string]string
}
```

- define NewClient function


- create interface for each api service with functions to call each endpoint
```go
type Client struct {
 	// ....
 
	// Services used for communicating with the API
	Tag           TagsService
}
```

## Define NewRequest
```go
// NewRequest creates an API request.
func (c *Client) NewRequest(ctx context.Context, method, urlStr string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		req, err = http.NewRequest(method, u.String(), nil)
		if err != nil {
			return nil, err
		}
	case http.MethodPost:
		buf := new(bytes.Buffer)
		if body != nil {
			err = json.NewEncoder(buf).Encode(body)
			if err != nil {
				return nil, err
			}
		}

		req, err = http.NewRequest(method, u.String(), buf)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", mediaType)
	default:
		//
	}
	// add headers
	req.Header.Set("Accept", mediaType)
	req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}
```
## Define Do client function where request is executed
```go
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {

	req = req.WithContext(ctx)
	resp := client.Do(req)


	response := newResponse(resp)

	// only when checking limiter rates in header
	c.ratemtx.Lock()
	c.Rate = response.Rate
	c.ratemtx.Unlock()

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}
}
```

## client error handling
	- define response error to parse any error message or error code returned by api
```go

// An ErrorResponse reports the error caused by an API request
type ErrorResponse struct {
	// HTTP response that caused this error
	Response *http.Response

	// Error message
	Message string `json:"message"`

	// RequestID returned from the API, useful to contact support.
	Code string `json:"code"`
}



```

## Check response errors
```go
// CheckResponse checks the API response for errors, and returns them if present. A response is considered an
// error if it has a status code outside the 200 range. API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse. Any other response body will be silently ignored.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && len(data) > 0 {
		err := json.Unmarshal(data, errorResponse)
		if err != nil {
			errorResponse.Message = string(data)
		}
	}

	return errorResponse
}

```



## create a new file for each service interface, e.g. tag.go

- create object that is being marshalled
- make sure to include annotation with json

```go
type ResourceType string

const (
	// DropletResourceType holds the string representing our ResourceType of Droplet.
	DropletResourceType ResourceType = "droplet"
	// ImageResourceType holds the string representing our ResourceType of Image.
	ImageResourceType ResourceType = "image"
)

type Resource struct {
	ID   string       `json:"resource_id,omitempty"`
	Type ResourceType `json:"resource_type,omitempty"`
}

type Tag struct {
	Name      string           `json:"name,omitempty"`
	Resources []*Resources `json:"resources,omitempty"`
}
```
	- 
- create
	* make sure you have context in every function
	  you need to be able to cancel api call upstream
	* if there are any options include them as arguments
	* return marshell output and 

	```go tag.go

	type TagsService interface {
		List(context.Context, *ListOptions) ([]Tag, *Response, error)
		Get(context.Context, string) (*Tag, *Response, error)
		Create(context.Context, *TagCreateRequest) (*Tag, *Response, error)
		Delete(context.Context, string) (*Response, error)
	}
```
- create service structure implementing service interface and instance of that service interface
```go
type TagsServiceOp {
	client *Client
}


```

- implement interface functions
	- create basepath for this service
	- !!! include context
```go
// Get a single tag

const tagsBasePath = "v2/tags"

func (s *TagsService) Get(ctx context.Context, name string) (*Tag, *Response, error) {
	path := fmt.Sprintf("%s/%s", tagsBasePath, name)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(tagRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Tag, resp, err
}
```
- create test file, tags_test.go





- use context in every endpoint function


## rate limiter
 



# Interview

When asked to integrate with API
- Read documentation of the endpoint; talk about division between services; possible errors;
- Start building it step by step using tests for 
- Create a new project and a folder for your restful api client
- Create main file and start with tests
- Clinet/Service, Service Interface and implementation
- Context in every service method to cancel any calls
- options object pagination 
- Request generator which populates headers related to authentication, accepted content, etc; as well as contruct parameters in the path based on passed options
- Response processor taking care of errors, extra header fields (e.g. rate limiter) and parsing/decoding body
- error handling, 200 <= code <= 299, no error; otherwise custom errors or default Response error
- rate limiter; 



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


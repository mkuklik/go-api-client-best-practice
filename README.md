# How to write Go RESTful API client
Here is a collection of best practices and an example of client library. I put this together while preparing for the interviews. It is easy to forget about some key patterns, which indicates the quality of Golang engineer


## Overall structure of API client

Client structure contains reusable components like http.Client and configuration. Api endpoints are packaged into Service structure, which is a member of a Client.

Service consists of an interface with endpoint functions and Service structure which implements this interface.

Requests are processed the following way. There is a client level function to create a request, which is passed to http.Client. This top level request generator takes care of common headers like authentication, Accept header, User-agent header, etc. It is also responsible for adding baseURL to the endpoint path.

Request is executed by Do function, which is at the Client level. It takes Request and interface for marshalled response body.
Before it calls http.Client Do function, it checks rate limits from the previous response. If limits are binding, it returns a fake response with 429 status error. Rate limiter values are stored in a Client structure and accessed with mutex lock. Response values from http.Client. Do goes through the following process:
- create new response with extra stuff (rate limits, headers, etc)
- save rate in client
- check for errors
- if there is no error, decode body to provided output interface


### Response processing
Initial response processing is taken care of at Client level. http.Response is wrapped by your own Response structure. You can keep there extra stuff like paginating variables (offset, limit, etc) and limiter rates in the response (parsed http headers  RateLimit-Limit, RateLimit-Remaining, RateLimit-Reset).
NewResponse function takes http.Client response as argument, parses response headers and populates this extra stuff in the response wrapper structure.

If there are some specific http errors which you want to cover in your application, e.g. rate limits (RateLimitError) or forbidden (AuthError), define such errors explicitly; for all other errors define a generic error, i.e. ErrorResponse structure (make sure to define Error() to satisfy Go error interface). For your convenience make sure to include reference to http.Response in your error definition. Http errors are covered the following way:

- Http status >= 200 and < 299 generate no error
- 202 Accepted Response is treated separately; 202 is when request was accepted but not yet processed. If your application acts on 202, define AcceptedError and return it on a successful 202 call.
- if http status >=300, then
   - read and unmarshal response body if read body != nil, in case API has message in the response
   - filter through status codes for which you want to generate custom Error
   - for any other error, generate default ResponseError which refers to http.Request

You don't want to flood the server with requests when the rate limiter is triggered; instead back off with request by returning a custom error, e.g. RateLimitError. Well-built API server should include rate limits in the response headers: ratelimit-limit, ratelimit-Remaining, ratelimit-reset or x-rate-*. These headers are processed, extracted from the response and saved in the Client; make sure to use mutex lock on r/w for these variables. Then on every request check whether limits are not binding and only then send the request. Also note that other errors like 403 Forbidden can be returned when reaching the limit. Thus

TODO:
- sanitizeURL

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

Create interface for each api service with functions to call each endpoint
```go
type Client struct {
 	// ....
 
	// Services used for communicating with the API
	Tag           TagsService
}
```



Define NewClient function
```go

const (
	defaultBaseURL = "https://api.foo.com/"
)


func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	baseURL, _ := url.Parse(defaultBaseURL)

	c := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent}
	c.Account = &TagsServiceOp{client: c}

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
## Do function

To send the request we define a `Do` function on the Client. It takes a context, a request and a pointer to value where decoded JSON in the body will be stored. `v` has to implement io.Writer. `Do` also checks for the errors in the response. If response contains an error, `Do` returns response and decodes JSON from the body, which is stored in the value pointed to by `v`.

- Make sure to close the Body of the request in order to reuse the TCP connection. Ensure to fully read the body before closing. If body is not emptied, connection won't be reused.

```go
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {

	req = req.WithContext(ctx)
	resp := client.Do(req)

	defer func() {
		// to reuse to connection read the body before closing
		const maxBodySlurpSize = 2 << 10
		if resp.ContentLength == -1 || resp.ContentLength <= maxBodySlurpSize {
			io.CopyN(ioutil.Discard, resp.Body, maxBodySlurpSize)
		}

		if rerr := resp.Body.Close(); err == nil {
			err = rerr
		}
	}()	


	response := newResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}
}
```

## Client error handling

If response contains an error, client returns an error structure which contains a pointer to http.Response and a message parsed from the JSON in the body of the response. 
```go

// An ErrorResponse reports the error caused by an API request
type ErrorResponse struct {
	// HTTP response that caused this error
	Response *http.Response

	// Error message
	Message string `json:"message"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v",
		r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.Message)
}
```

CheckResponse function checks the API errors and return them if present.
- No error is returned when status is within 200 range. 
- A JSON response body maps to ErrorResponse if present; it is silently ignored. otherwise.


```go
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

## Custom errors

If you want to treat some api errors in special way, define a custom error and return it in `CheckResponse` 

```go

type RateLimitError struct {
	Response *http.Response // HTTP response that caused this error

	Rate     Rate           // Rate specifies last known rate limit for the client
	Message  string         `json:"message"` // error message
}

func (r *RateLimitError) Error() string {
	return fmt.Sprintf("%v %v: %d %v %v",
		r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
		r.Response.StatusCode, r.Message, formatRateReset(time.Until(r.Rate.Reset.Time)))
}


func CheckResponse(r *http.Response) error {


	// ...


	// Re-populate error response body because GitHub error responses are often
	// undocumented and inconsistent.

	r.Body = ioutil.NopCloser(bytes.NewBuffer(data))

	switch {
	// ....
	case r.StatusCode == http.StatusForbidden && r.Header.Get(headerRateRemaining) == "0":
		return &RateLimitError{
			Rate:     parseRate(r),
			Response: errorResponse.Response,
			Message:  errorResponse.Message,
		}
	// ...
	default:
		return errorResponse
	}
}


```

You can check for these errors using type assertion.

```go
v, ok := target.(*RateLimitError)
if !ok {
	return false
}

```


## Define service

Define structures, which JSON in a response body is unmarshalled into. Make sure to include tags for json, xml, etc

```go tag.go
type ResourceType string

const (
	DropletResourceType ResourceType = "droplet"
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

Define service interface 
* make sure you have context in every function you need to be able to cancel api call upstream
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

Create a structure, which implements the service interface. Pointer to http.Client is passed into it.
```go
type TagsServiceOp {
	client *Client
}
```

Implement interface functions. It should create request with `NewRequest` and modify it accordingly. Then allocate empty value for response data and call Do to send a request.
```go

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

## Testing
- create test file, tags_test.go
```go


```




## Pagination
TODO



## Rate limiter
Create a structure with rates limits, which is stored in the Client. Rate will be populated from every responce. Since client can process many requests at the same time, we need a lock whenever we read/write to `Rate`
```go
type Client struct {
	// ...

	Rate    Rate
	ratemtx sync.Mutex

	// ...
}
```

In out response we will add rate and parse it when creating a new `Response` from http.Response. Note that header names listed below can be found in RAte Limiter RFC Draft. Some APIs use X-Rate-Limit-Limit, X-Rate-Limit-Remaining,  X-Rate-Limit-Reset.

```go

const (
	headerRateLimit     = "RateLimit-Limit"
	headerRateRemaining = "RateLimit-Remaining"
	headerRateReset     = "RateLimit-Reset"
)

func newResponse(r *http.Response) *Response {
	response := Response{Response: r}
	
	parseRate(response)

	return &response
}

func parseRate(r *Response) {

	if limit := r.Header.Get(headerRateLimit); limit != "" {
		response.Rate.Limit, _ = strconv.Atoi(limit)
	}
	if remaining := r.Header.Get(headerRateRemaining); remaining != "" {
		response.Rate.Remaining, _ = strconv.Atoi(remaining)
	}
	if reset := r.Header.Get(headerRateReset); reset != "" {
		if v, _ := strconv.ParseInt(reset, 10, 64); v != 0 {
			response.Rate.Reset = Timestamp{time.Unix(v, 0)}
		}
	}
}
```

Now `Rate` in the Client is populated when running Do

```go

func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {


	// If we've hit rate limit, don't make further requests before Reset time.
	if err := c.checkRateLimitBeforeDo(req); err != nil {
		return &Response{
			Response: err.Response,
			Rate:     err.Rate,
		}, err
	}


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

Finally, we need to check whether rate limits are binding and return error.

```go
// RateLimitError occurs when GitHub returns 403 Forbidden response with a rate limit
// remaining value of 0.
type RateLimitError struct {
	Rate     Rate           // Rate specifies last known rate limit for the client
	Response *http.Response // HTTP response that caused this error
	Message  string         `json:"message"` // error message
}

func (r *RateLimitError) Error() string {
	return fmt.Sprintf("%v %v: %d %v %v",
		r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
		r.Response.StatusCode, r.Message, formatRateReset(time.Until(r.Rate.Reset.Time)))
}


func (c *Client) checkRateLimitBeforeDo(req *http.Request) *RateLimitError {

	c.ratemtx.Lock()
	rate := c.Rate
	c.ratemtx.Unlock()
	if !rate.Reset.Time.IsZero() && rate.Remaining == 0 && time.Now().Before(rate.Reset.Time) {
		// Create a fake response.
		resp := &http.Response{
			Status:     http.StatusText(http.StatusForbidden),
			StatusCode: http.StatusForbidden,
			Request:    req,
			Header:     make(http.Header),
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}
		return &RateLimitError{
			Rate:     rate,
			Response: resp,
			Message:  fmt.Sprintf("API rate limit of %v still exceeded until %v, not making remote request.", rate.Limit, rate.Reset.Time),
		}
	}

	return nil
}
```

## Test everything 
TODO what to test



# !!! Things to remember !!!

When asked to integrate with API
- Read documentation of the endpoint; talk about division between services; possible errors;
- Start building it step by step using tests along the way. Make sure every step is working and build on top of it
- Create a new project and a folder for your restful api client
- Create main file and start with tests
- Client/Service pattern, Service Interface and implementation
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


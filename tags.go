package client

import (
	"context"
	"net/http"
)

/*  Objects */
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
	Name      string      `json:"name,omitempty"`
	Resources []*Resource `json:"resources,omitempty"`
}

/* SERVICE */

// TagsService is an interface for interfacing with the tags
// endpoints of the DigitalOcean API
type TagsService interface {
	List(context.Context, *ListOptions) ([]Tag, *Response, error)
	Get(context.Context, string) (*Tag, *Response, error)
	Create(context.Context, string) (*Tag, *Response, error)
	Delete(context.Context, string) (*Response, error)
}

// TagsServiceOp handles communication with tag related method of the
// DigitalOcean API.
type TagsServiceOp struct {
	client *Client
}

var _ TagsService = &TagsServiceOp{}

// List all tags
func (s *TagsServiceOp) List(ctx context.Context, opt *ListOptions) ([]Tag, *Response, error) {
	path := tagsBasePath
	path, err := addOptions(path, opt)

	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(tagsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if l := root.Links; l != nil {
		resp.Links = l
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Tags, resp, err
}

// Get a single tag
func (s *TagsServiceOp) Get(ctx context.Context, name string) (*Tag, *Response, error) {
	return nil, nil, nil
}

// Create a new tag
func (s *TagsServiceOp) Create(ctx context.Context, some string) (*Tag, *Response, error) {
	return nil, nil, nil
}

// Delete an existing tag
func (s *TagsServiceOp) Delete(ctx context.Context, some string) (*Response, error) {
	return nil, nil
}

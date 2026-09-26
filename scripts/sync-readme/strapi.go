package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Icon struct {
	Name  string `json:"iconName"`
	Color string `json:"color"`
}

type Tag struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Level    int    `json:"level"`
	Featured bool   `json:"featured"`
	Icon     *Icon  `json:"icon"`
}

type Project struct {
	DocumentID string `json:"documentId"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Type       string `json:"projectType"`
	Status     string `json:"projectStatus"`
	Difficulty string `json:"projectDifficulty"`
	Preview    string `json:"preview"`
	GitHub     string `json:"github"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Featured   bool   `json:"featured"`
	Tags       []Tag  `json:"tags"`
}

type Experience struct {
	Title          string `json:"title"`
	Company        string `json:"company"`
	Description    string `json:"description"`
	Location       string `json:"location"`
	EmploymentType string `json:"employmentType"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	URL            string `json:"url"`
	Featured       bool   `json:"featured"`
}

type Education struct {
	Title       string `json:"title"`
	Institution string `json:"institution"`
	Grade       string `json:"grade"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	URL         string `json:"url"`
	Featured    bool   `json:"featured"`
}

type client struct {
	base  string
	token string
	http  *http.Client
}

func newClient(base, token string) *client {
	return &client{base: base, token: token, http: &http.Client{Timeout: 10 * time.Second}}
}

// Errors carry the path and status only: never the token, never the body.
func (c *client) get(ctx context.Context, path string, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", path, res.Status)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 16<<20)).Decode(out); err != nil {
		return fmt.Errorf("GET %s: decode: %w", path, err)
	}
	return nil
}

type page[T any] struct {
	Data []T `json:"data"`
	Meta struct {
		Pagination struct {
			PageCount int `json:"pageCount"`
		} `json:"pagination"`
	} `json:"meta"`
}

// Strapi caps pageSize at 100 by default, so larger collections need paging.
func fetchAll[T any](ctx context.Context, c *client, path string, q url.Values) ([]T, error) {
	var all []T
	for p := 1; p <= 20; p++ {
		q.Set("pagination[page]", strconv.Itoa(p))
		q.Set("pagination[pageSize]", "100")
		var pg page[T]
		if err := c.get(ctx, path, q, &pg); err != nil {
			return nil, err
		}
		all = append(all, pg.Data...)
		if p >= pg.Meta.Pagination.PageCount {
			break
		}
	}
	return all, nil
}

func query(pairs ...string) url.Values {
	q := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		q.Add(pairs[i], pairs[i+1])
	}
	return q
}

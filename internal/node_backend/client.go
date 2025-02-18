package node_backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	jsoniter "github.com/json-iterator/go"

	"github.com/deweb-services/gateway-st/internal/domain"
)

const nodeBucketPath = "/access-key"

var (
	ErrInternal     = errors.New("internal error")
	ErrAlreadyExist = errors.New("already exist")
)

type Client struct {
	hc *http.Client

	host  string
	token string
}

func New(hc *http.Client, host, token string) *Client {
	return &Client{
		hc:    hc,
		host:  host,
		token: token,
	}
}

type createPayload struct {
	ProjectUUID string `json:"projectUUID"`
	BucketName  string `json:"bucketName"`
}

func (c *Client) Create(ctx context.Context, projectUUID, bucketName string) (*domain.AccessKey, error) {
	ar, err := jsoniter.Marshal(createPayload{ProjectUUID: projectUUID, BucketName: bucketName})
	if err != nil {
		return nil, fmt.Errorf("marshal request body error: %w", err)
	}

	u, err := url.JoinPath(c.host, nodeBucketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to construct url: %w", err)
	}

	code, err := c.do(ctx, u, http.MethodPost, bytes.NewBuffer(ar), map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, ErrInternal
	}

	if code >= 500 {
		return nil, ErrInternal
	} else if code >= 400 && code < 500 {
		return nil, ErrAlreadyExist
	}

	return &domain.AccessKey{}, nil
}

type deletePayload struct {
	ProjectUUID string `json:"projectUUID"`
	AccessKey   string `json:"accessKey"`
	SecretID    string `json:"secretID"`
}

func (c *Client) Revoke(ctx context.Context, projectUUID, accessKey, secretID string) error {
	dr, err := jsoniter.Marshal(deletePayload{ProjectUUID: projectUUID, AccessKey: accessKey, SecretID: secretID})
	if err != nil {
		return fmt.Errorf("marshal request body error: %w", err)
	}

	u, err := url.JoinPath(c.host, nodeBucketPath)
	if err != nil {
		return fmt.Errorf("failed to construct url: %w", err)
	}

	code, err := c.do(ctx, u, http.MethodDelete, bytes.NewBuffer(dr), nil)
	if err != nil || (code >= 300 || code < 200) {
		return ErrInternal
	}

	return nil
}

func (c *Client) do(
	ctx context.Context,
	url string,
	method string,
	reader io.Reader,
	headers map[string]string,
) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not create a request: %w", err)
	}

	req.Header.Set("Authorization", c.token)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not do request: %w", err)
	}

	if _, err := io.ReadAll(resp.Body); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not read response body: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return resp.StatusCode, nil
}

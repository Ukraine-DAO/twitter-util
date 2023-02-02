package twitterutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type MediaCategory string

const (
	MediaCategoryTweetImage   = "tweet_image"
	MediaCategoryAmplifyVideo = "amplify_video"
	MediaCategoryTweetGif     = "tweet_gif"
	MediaCategoryTweetVideo   = "tweet_video"
)

// MediaUpload uploads a media to Twitter and returns a media ID. `client` is
// responsible for authentication.
func MediaUpload(ctx context.Context, client HttpClient, media []byte, category MediaCategory) (string, error) {
	payload := &bytes.Buffer{}
	m := multipart.NewWriter(payload)
	w, err := m.CreateFormFile("media", "image.png")
	if err != nil {
		return "", fmt.Errorf("creating a multipart payload: %w", err)
	}
	if _, err := w.Write(media); err != nil {
		return "", fmt.Errorf("writing file content into payload: %w", err)
	}
	if err := m.Close(); err != nil {
		return "", fmt.Errorf("finalizing the payload: %w", err)
	}

	query := url.Values{}
	query.Set("media_category", string(category))
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://upload.twitter.com/1.1/media/upload.json?%s", query.Encode()), payload)
	if err != nil {
		return "", fmt.Errorf("creating request object: %w", err)
	}

	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data, boundary=%q", m.Boundary()))
	req.Header.Set("Content-Length", fmt.Sprint(payload.Len()))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending upload request: %w", err)
	}
	defer resp.Body.Close()

	var v struct {
		ID  string `json:"media_id_string"`
		Key string `json:"media_key"`
	}

	if resp.StatusCode >= 400 {
		log.Printf("%+v", resp)
		b, _ := io.ReadAll(resp.Body)
		log.Printf("%s", string(b))
		return "", fmt.Errorf("upload request returned an error: %d %s\n%s", resp.StatusCode, resp.Status, string(b))
	}

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return v.ID, nil
}

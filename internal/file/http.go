package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func httpLastModified(res *http.Response) (time.Time, error) {
	t, err := time.Parse(time.RFC1123, res.Header.Get("Last-Modified"))
	if err != nil {
		return zeroTime, fmt.Errorf("error parsing Last-Modified header: %v", err)
	}
	return t, nil
}

func readFileFromHTTP(ctx context.Context, fileURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fileURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got error while retrieving HTTP object, http status: %s, status code: %d", res.Status, res.StatusCode)
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func httpModTime(ctx context.Context, fileURL string) (time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", fileURL, nil)
	if err != nil {
		return zeroTime, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return zeroTime, err
	}

	if res.StatusCode != http.StatusOK {
		return zeroTime, fmt.Errorf("got error while retrieving HTTP object, http status: %s, status code: %d", res.Status, res.StatusCode)
	}

	defer res.Body.Close()
	return httpLastModified(res)
}

package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Forward relays an incoming request to the provided upstream base URL and
// writes the upstream response back to the client. The suffix is appended to the
// base URL and should begin with a slash when targeting nested resources.
func Forward(w http.ResponseWriter, r *http.Request, client *http.Client, base string, suffix string) {
	target, err := buildTargetURL(base, suffix, r.URL.RawQuery)
	if err != nil {
		http.Error(w, "invalid upstream url", http.StatusBadGateway)
		return
	}

	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusBadGateway)
		return
	}

	copyRequestHeaders(req.Header, r.Header)

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	if resp.Body != nil {
		io.Copy(w, resp.Body)
	}
}

func buildTargetURL(base string, suffix string, rawQuery string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	if suffix != "" {
		if !strings.HasSuffix(baseURL.Path, "/") {
			baseURL.Path += "/"
		}
		baseURL.Path = strings.TrimSuffix(baseURL.Path, "//")
		baseURL.Path = strings.TrimSuffix(baseURL.Path, "/") + suffix
	}

	if rawQuery != "" {
		baseURL.RawQuery = rawQuery
	}

	return baseURL.String(), nil
}

func readRequestBody(r *http.Request) (io.ReadCloser, error) {
	if r.Body == nil {
		return http.NoBody, nil
	}
	defer r.Body.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r.Body); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func copyRequestHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		lower := strings.ToLower(key)
		if lower == "content-type" || strings.HasPrefix(lower, "x-") {
			for _, value := range values {
				dst.Add(key, value)
			}
		}
	}
}

func copyResponseHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		lower := strings.ToLower(key)
		if lower == "content-type" || lower == "content-length" || strings.HasPrefix(lower, "x-") {
			dst[key] = append([]string(nil), values...)
		}
	}
}

// DecodeJSONArray normalizes upstream payloads that may either be a bare array
// or a paginated envelope (with a top-level "results" field). It returns an
// empty slice when decoding fails.
func DecodeJSONArray(payload []byte) []map[string]any {
	var arr []map[string]any
	if err := json.Unmarshal(payload, &arr); err == nil {
		return arr
	}

	var envelope struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil && envelope.Results != nil {
		return envelope.Results
	}
	return []map[string]any{}
}

// DecodeObject parses payload into a map. When decoding fails it returns an
// empty map and the encountered error to allow callers to fall back to defaults.
func DecodeObject(payload []byte) (map[string]any, error) {
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return map[string]any{}, err
	}
	return obj, nil
}

// ReadBody extracts the full body from an HTTP response, ensuring a nil body is
// treated as an empty payload.
func ReadBody(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return []byte{}, nil
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// DrainAndClose discards any remaining bytes to allow connection reuse.
func DrainAndClose(resp *http.Response) {
	if resp.Body == nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// ErrBadGateway is returned when the upstream response cannot be consumed.
var ErrBadGateway = errors.New("bad gateway")

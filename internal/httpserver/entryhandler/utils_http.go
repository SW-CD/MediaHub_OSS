package entryhandler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// writeJSONFileResponseStream streams a binary io.Reader as a Base64-encoded Data URI JSON response directly to w.
// This eliminates buffering full files in memory.
func writeJSONFileResponseStream(w io.Writer, filename, mimeType string, reader io.Reader) error {
	fnBytes, err := json.Marshal(filename)
	if err != nil {
		return err
	}
	mtBytes, err := json.Marshal(mimeType)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf(`{"filename":%s,"mime_type":%s,"data":"data:%s;base64,`, string(fnBytes), string(mtBytes), mimeType)
	if _, err := io.WriteString(w, prefix); err != nil {
		return err
	}

	b64Writer := base64.NewEncoder(base64.StdEncoding, w)
	if _, err := io.Copy(b64Writer, reader); err != nil {
		b64Writer.Close()
		return err
	}
	if err := b64Writer.Close(); err != nil {
		return err
	}

	if _, err := io.WriteString(w, `"}`); err != nil {
		return err
	}

	return nil
}

// parseRange parses a standard HTTP Range header (e.g. "bytes=1000-2000")
// and returns the offset and length relative to the fileSize.
func parseRange(header string, fileSize int64) ([]byteRange, error) {
	if !strings.HasPrefix(header, "bytes=") {
		return nil, fmt.Errorf("invalid unit")
	}

	var ranges []byteRange
	chk := strings.TrimPrefix(header, "bytes=")
	parts := strings.Split(chk, ",") // Handle multiple ranges "0-50, 100-150"

	for _, part := range parts {
		part = strings.TrimSpace(part)
		bounds := strings.Split(part, "-")
		if len(bounds) != 2 {
			continue
		}

		var start, end int64
		var err error

		if bounds[0] == "" {
			// suffix-byte-range-spec: "-500" (Last 500 bytes)
			suffix, err := strconv.ParseInt(bounds[1], 10, 64)
			if err != nil {
				return nil, err
			}
			if suffix > fileSize {
				suffix = fileSize
			}
			start = fileSize - suffix
			end = fileSize - 1
		} else {
			start, err = strconv.ParseInt(bounds[0], 10, 64)
			if err != nil {
				return nil, err
			}

			if bounds[1] == "" {
				// "100-" (From 100 to end)
				end = fileSize - 1
			} else {
				// "100-200"
				end, err = strconv.ParseInt(bounds[1], 10, 64)
				if err != nil {
					return nil, err
				}
			}
		}

		// Validation
		if start < 0 {
			start = 0
		}
		if end >= fileSize {
			end = fileSize - 1
		}
		if start > end {
			return nil, fmt.Errorf("invalid range")
		}

		ranges = append(ranges, byteRange{
			start:  start,
			length: end - start + 1,
		})
	}
	return ranges, nil
}

// encodeReaderAsJSON reads data from an io.Reader stream, encodes it as Base64,
// and returns it as a JSON object.
// This is used to support clients that cannot handle binary streams with auth headers.
func encodeReaderAsJSON(reader io.Reader, filename, mimeType string) (FileJSONResponse, error) {
	// 1. Read the stream into memory
	// io.ReadAll reads from the reader until an error or EOF and returns the data it read.
	data, err := io.ReadAll(reader)
	if err != nil {
		return FileJSONResponse{}, err
	}

	// 2. Encode to Base64
	b64Str := base64.StdEncoding.EncodeToString(data)

	// 3. Construct the response object
	resp := FileJSONResponse{
		Filename: filename,
		MimeType: mimeType,
		// Format strictly follows the Data URI scheme: data:[<mediatype>][;base64],<data>
		Data: fmt.Sprintf("data:%s;base64,%s", mimeType, b64Str),
	}

	return resp, nil
}

// parseQueryInt safely parses an integer from query parameters, falling back to a default value if omitted.
func parseQueryInt(r *http.Request, key string, defaultValue int) (int, error) {
	if val := r.URL.Query().Get(key); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("invalid value for parameter '%s': must be an integer", key)
		}
		return parsed, nil
	}
	return defaultValue, nil
}

// parseQueryInt64 safely parses a 64-bit integer from query parameters, falling back to a default value if omitted.
func parseQueryInt64(r *http.Request, key string, defaultValue int64) (int64, error) {
	if val := r.URL.Query().Get(key); val != "" {
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid value for parameter '%s': must be an integer", key)
		}
		return parsed, nil
	}
	return defaultValue, nil
}

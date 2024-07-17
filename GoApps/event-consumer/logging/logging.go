package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
)

type LoggableRequest struct {
	Method           string      `json:"method,omitempty"`
	URL              *url.URL    `json:"URL,omitempty"`
	Proto            string      `json:"proto,omitempty"`
	ProtoMajor       int         `json:"protoMajor,omitempty"`
	ProtoMinor       int         `json:"protoMinor,omitempty"`
	Header           http.Header `json:"headers,omitempty"`
	Body             string      `json:"body,omitempty"`
	ContentLength    int64       `json:"contentLength,omitempty"`
	TransferEncoding []string    `json:"transferEncoding,omitempty"`
	Host             string      `json:"host,omitempty"`
	Trailer          http.Header `json:"trailer,omitempty"`
	RemoteAddr       string      `json:"remoteAddr"`
	RequestURI       string      `json:"requestURI"`
}

func LogRequest(req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println("⚠️ Failed to read request body")
	}
	_ = req.Body.Close()
	// Replace the body with a new reader after reading from the original
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	b, err := json.MarshalIndent(toReq(req), "", "  ")
	if err != nil {
		log.Println("⚠️ Failed to marshal request:", err)
	}

	log.Println(string(b))
}

func toReq(req *http.Request) LoggableRequest {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println("⚠️ Failed to read request body")
	}
	_ = req.Body.Close()
	// Replace the body with a new reader after reading from the original
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	return LoggableRequest{
		Method:           req.Method,
		URL:              req.URL,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Header:           req.Header,
		Body:             string(body),
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Host:             req.Host,
		Trailer:          req.Trailer,
		RemoteAddr:       req.RemoteAddr,
		RequestURI:       req.RequestURI,
	}
}

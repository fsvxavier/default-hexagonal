package nethttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Requester struct {
	Client               *http.Client
	Headers              map[string]string
	BaseURL              string
	StructUnmarshal      any
	getConn              time.Time
	dnsStart             time.Time
	dnsDone              time.Time
	connectDone          time.Time
	tlsHandshakeStart    time.Time
	tlsHandshakeDone     time.Time
	gotConn              time.Time
	gotFirstResponseByte time.Time
	endTime              time.Time
	gotConnInfo          httptrace.GotConnInfo
}

type Response struct {
	Body       []byte
	StatusCode int
	IsError    bool
}

type IHttpRequester interface {
	Get(ctx context.Context, endpoint string) (*Response, error)
	Post(ctx context.Context, endpoint string, body []byte) (*Response, error)
	Put(ctx context.Context, endpoint string, body []byte) (*Response, error)
	Delete(ctx context.Context, endpoint string) (*Response, error)
	SetHeaders(headers map[string]string) *Requester
	SetBaseURL(baseURL string) *Requester
	Unmarshal(v any) *Requester
}

const (
	REQ_MAX_IDLE_CONNS          = "1000"
	REQ_MAX_IDLE_CONNS_PER_HOST = "1000"
	REQ_MAX_CONNS_PER_HOST      = "2000"
	REQ_IDLE_CONN_TIMEOUT       = "3600"
	REQ_TLS_ENABLE              = false
	REQ_TRACE_ENABLE            = false
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// SetHeaders method sets multiple headers field and its values at one go in the client instance.
// These headers will be applied to all requests raised from this client instance. Also it can be
// overridden at request level headers options.
// For Example: To set `Content-Type` and `Accept` as `application/json`
//
//	request.SetHeaders(map[string]string{
//			"Content-Type": "application/json",
//			"Accept": "application/json",
//		})
func (r *Requester) SetHeaders(headers map[string]string) *Requester {
	r.Headers = headers
	return r
}

// SetErrorHandler method is to register the response `ErrorHandler` for current `Request`.
func (r *Requester) SetBaseURL(baseURL string) *Requester {
	r.BaseURL = baseURL
	return r
}

// Post method performs the HTTP POST request for current `Request`.
func (r *Requester) Post(ctx context.Context, endpoint string, body []byte) (*Response, error) {
	return r.Execute(ctx, fiber.MethodPost, endpoint, bytes.NewBuffer(body))
}

// Get method performs the HTTP GET request for current `Request`.
func (r *Requester) Get(ctx context.Context, endpoint string) (*Response, error) {
	return r.Execute(ctx, fiber.MethodGet, endpoint, nil)
}

// Put method performs the HTTP PUT request for current `Request`.
func (r *Requester) Put(ctx context.Context, endpoint string, body []byte) (*Response, error) {
	return r.Execute(ctx, fiber.MethodPut, endpoint, bytes.NewBuffer(body))
}

// Delete method performs the HTTP DELETE request for current `Request`.
func (r *Requester) Delete(ctx context.Context, endpoint string) (*Response, error) {
	return r.Execute(ctx, fiber.MethodDelete, endpoint, nil)
}

// Unmarshal method unmarshals the HTTP response body to given struct.
func (r *Requester) Unmarshal(v any) *Requester {
	r.StructUnmarshal = v
	return r
}

func (r *Requester) Execute(ctx context.Context, method, url string, body io.Reader) (response *Response, err error) {
	ddSpan, ok := tracer.SpanFromContext(ctx)
	defer ddSpan.Finish()

	if ok {
		err := tracer.Inject(ddSpan.Context(), tracer.TextMapCarrier(r.Headers))
		if err != nil {
			return nil, err
		}
	}

	tracer := &httptrace.ClientTrace{
		DNSStart: func(dnsstartInfo httptrace.DNSStartInfo) {
			r.dnsStart = time.Now()
		},
		DNSDone: func(dnsinfo httptrace.DNSDoneInfo) {
			r.dnsDone = time.Now()
		},
		ConnectStart: func(network, addr string) {
			if r.dnsDone.IsZero() {
				r.dnsDone = time.Now()
			}
			if r.dnsStart.IsZero() {
				r.dnsStart = r.dnsDone
			}
		},
		ConnectDone: func(net, addr string, err error) {
			r.connectDone = time.Now()
		},
		GetConn: func(hostPort string) {
			r.getConn = time.Now()
		},
		GotConn: func(ci httptrace.GotConnInfo) {
			r.gotConn = time.Now()
			r.gotConnInfo = ci
		},
		GotFirstResponseByte: func() {
			r.gotFirstResponseByte = time.Now()
		},
		TLSHandshakeStart: func() {
			r.tlsHandshakeStart = time.Now()
		},
		TLSHandshakeDone: func(tlscon tls.ConnectionState, errr error) {
			r.tlsHandshakeDone = time.Now()
		},
	}

	uriREquest := r.BaseURL + url

	reqs, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, tracer), method, uriREquest, body)
	if err != nil {
		return nil, err
	}

	reqs.Close = false

	if r.Headers != nil {
		for k, v := range r.Headers {
			reqs.Header.Set(k, v)
		}
	}

	reqs.Header.Set("Content-Type", "application/json")

	isErrors := false
	resp, err := r.Client.Do(reqs)
	if err != nil {
		isErrors = true
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// read response body
	if resp.StatusCode == http.StatusOK {
		if r.StructUnmarshal != nil {
			err := json.Unmarshal(respBody, r.StructUnmarshal)
			if err != nil {
				return nil, err
			}
		}

		response = &Response{
			Body:       respBody,
			StatusCode: resp.StatusCode,
			IsError:    isErrors,
		}
	} else {
		var respError error
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			respError = fmt.Errorf("%d-%s", resp.StatusCode, string(respBody))
		}
		return nil, respError
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	defaultRecTraceEnable := REQ_TRACE_ENABLE
	if os.Getenv("REQ_TRACE_ENABLE") != "" {
		defaultRecTraceEnable = (os.Getenv("REQ_TRACE_ENABLE") == "true")
	}
	if defaultRecTraceEnable {
		ti := r.TraceInfo()

		jsonTracer := `{"DNSLookup":"%v","URI":"%s","RemoteAddr":"%v","LocalAddr":"%v","ConnTime":"%v", "TCPConnTime":"%v",` +
			`"TLSHandshake":"%v","ServerTime":"%v","ResponseTime":"%v","TotalTime":"%v","IsConnReused":"%v","IsConnWasIdle":"%v",` +
			`"ConnIdleTime":"%v"}`

		fmt.Println(fmt.Sprintf(jsonTracer, ti.DNSLookup, uriREquest, ti.RemoteAddr, ti.LocalAddr, ti.ConnTime, ti.TCPConnTime, ti.TLSHandshake, ti.ServerTime, ti.ResponseTime, ti.TotalTime, ti.IsConnReused, ti.IsConnWasIdle, ti.ConnIdleTime))
	}
	return response, nil
}

func NewRequester(client *http.Client) *Requester {
	return &Requester{
		Client: client,
	}
}

func New() *http.Client {
	defaultMaxIdleConns := REQ_MAX_IDLE_CONNS
	if os.Getenv("REQ_MAX_IDLE_CONNS") != "" {
		defaultMaxIdleConns = os.Getenv("REQ_MAX_IDLE_CONNS")
	}
	maxIdleConns, err := strconv.Atoi(defaultMaxIdleConns)
	if err != nil {
		log.Fatalf("Erro to convert REQ_MAX_IDLE_CONNS %+v", err.Error())
	}

	defaultMaxIdleConnsPerHost := REQ_MAX_IDLE_CONNS_PER_HOST
	if os.Getenv("REQ_MAX_IDLE_CONNS_PER_HOST") != "" {
		defaultMaxIdleConnsPerHost = os.Getenv("REQ_MAX_IDLE_CONNS_PER_HOST")
	}
	maxIdleConnsPerHost, err := strconv.Atoi(defaultMaxIdleConnsPerHost)
	if err != nil {
		log.Fatalf("Erro to convert REQ_MAX_IDLE_CONNS_PER_HOST %+v", err.Error())
	}

	defaultMaxConnsPerHost := REQ_MAX_CONNS_PER_HOST
	if os.Getenv("REQ_MAX_CONNS_PER_HOST") != "" {
		defaultMaxConnsPerHost = os.Getenv("REQ_MAX_CONNS_PER_HOST")
	}
	maxConnsPerHost, err := strconv.Atoi(defaultMaxConnsPerHost)
	if err != nil {
		log.Fatalf("Erro to convert REQ_MAX_CONNS_PER_HOST %+v", err.Error())
	}

	defaultIdleConnTimeout := REQ_IDLE_CONN_TIMEOUT
	if os.Getenv("REQ_IDLE_CONN_TIMEOUT") != "" {
		defaultIdleConnTimeout = os.Getenv("REQ_IDLE_CONN_TIMEOUT")
	}
	idleConnTimeout, err := time.ParseDuration(defaultIdleConnTimeout + "s")
	if err != nil {
		log.Fatalf("Erro to convert REQ_IDLE_CONN_TIMEOUT %+v", err.Error())
	}

	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		MaxConnsPerHost:     maxConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
	}

	defaultTLSEnable := REQ_TLS_ENABLE
	if os.Getenv("REQ_TLS_ENABLE") != "" && os.Getenv("REQ_TLS_ENABLE") == "true" {
		defaultTLSEnable = true
	}

	if defaultTLSEnable {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client := &http.Client{
		Timeout:   idleConnTimeout,
		Transport: transport,
	}

	return client
}

// ‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// TraceInfo struct
// _______________________________________________________________________

// TraceInfo struct is used provide request trace info such as DNS lookup
// duration, Connection obtain duration, Server processing duration, etc.
//
// Since v2.0.0.
type TraceInfo struct {
	// DNSLookup is a duration that transport took to perform
	// DNS lookup.
	DNSLookup time.Duration

	// ConnTime is a duration that took to obtain a successful connection.
	ConnTime time.Duration

	// TCPConnTime is a duration that took to obtain the TCP connection.
	TCPConnTime time.Duration

	// TLSHandshake is a duration that TLS handshake took place.
	TLSHandshake time.Duration

	// ServerTime is a duration that server took to respond first byte.
	ServerTime time.Duration

	// ResponseTime is a duration since first response byte from server to
	// request completion.
	ResponseTime time.Duration

	// TotalTime is a duration that total request took end-to-end.
	TotalTime time.Duration

	// IsConnReused is whether this connection has been previously
	// used for another HTTP request.
	IsConnReused bool

	// IsConnWasIdle is whether this connection was obtained from an
	// idle pool.
	IsConnWasIdle bool

	// ConnIdleTime is a duration how long the connection was previously
	// idle, if IsConnWasIdle is true.
	ConnIdleTime time.Duration

	// RequestAttempt is to represent the request attempt made during a Resty
	// request execution flow, including retry count.
	RequestAttempt int

	// RemoteAddr returns the remote network address.
	RemoteAddr net.Addr

	// LocalAddr returns the local network address.
	LocalAddr net.Addr
}

func (r *Requester) TraceInfo() TraceInfo {
	if r == nil {
		return TraceInfo{}
	}

	ti := TraceInfo{
		DNSLookup:      r.dnsDone.Sub(r.dnsStart),
		TLSHandshake:   r.tlsHandshakeDone.Sub(r.tlsHandshakeStart),
		ServerTime:     r.gotFirstResponseByte.Sub(r.gotConn),
		IsConnReused:   r.gotConnInfo.Reused,
		IsConnWasIdle:  r.gotConnInfo.WasIdle,
		ConnIdleTime:   r.gotConnInfo.IdleTime,
		RemoteAddr:     r.gotConnInfo.Conn.RemoteAddr(),
		LocalAddr:      r.gotConnInfo.Conn.RemoteAddr(),
		RequestAttempt: 0,
	}

	// Calculate the total time accordingly,
	// when connection is reused
	if r.gotConnInfo.Reused {
		ti.TotalTime = r.endTime.Sub(r.getConn)
	} else {
		ti.TotalTime = r.endTime.Sub(r.dnsStart)
	}

	// Only calculate on successful connections
	if !r.connectDone.IsZero() {
		ti.TCPConnTime = r.connectDone.Sub(r.dnsDone)
	}

	// Only calculate on successful connections
	if !r.gotConn.IsZero() {
		ti.ConnTime = r.gotConn.Sub(r.getConn)
	}

	// Only calculate on successful connections
	if !r.gotFirstResponseByte.IsZero() {
		ti.ResponseTime = r.endTime.Sub(r.gotFirstResponseByte)
	}

	// Capture remote address info when connection is non-nil
	if r.gotConnInfo.Conn != nil {
		ti.RemoteAddr = r.gotConnInfo.Conn.RemoteAddr()
		ti.LocalAddr = r.gotConnInfo.Conn.LocalAddr()
	}

	return ti
}

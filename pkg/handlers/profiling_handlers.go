package handlers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

const (
	defaultCrioHost    = "localhost"
	defaultCrioPort    = "6060"
	defaultKubeletPort = "10250"
	crioProfilePath    = "debug/pprof/profile"
	kubeletProfilePath = "debug/pprof/profile"
)

// profileCrio triggers CRIO profiling on localhost.
func (h *Handlers) profileCrio(uid string) runs.ProfilingRun {
	client := &http.Client{
		Transport: newDefaultHTTPTransport().build(),
	}
	if h.CrioPreferUnixSocket {
		client.Transport = newDefaultHTTPTransport().withUnixDialContext(h.CrioUnixSocket).build()
	}

	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(defaultCrioHost, defaultCrioPort),
		Path:   crioProfilePath,
	}

	hlog.Infof("requesting CRIO profiling, runID: %s", uid)
	return sendHTTPProfileRequest(runs.CrioRun, "GET", u.String(), "", h.crioPprofOutputFilePath(uid), client)
}

// profileKubelet triggers Kubelet profiling on h.NodeIP using h.Token for authorization.
func (h *Handlers) profileKubelet(uid string) runs.ProfilingRun {
	client := &http.Client{
		Transport: newDefaultHTTPTransport().withRootCAs(h.CACerts).build(),
	}

	u := url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(h.NodeIP, defaultKubeletPort),
		Path:   kubeletProfilePath,
	}

	hlog.Infof("requesting Kubelet profiling, runID: %s", uid)
	return sendHTTPProfileRequest(runs.KubeletRun, "GET", u.String(), h.Token, h.kubeletPprofOutputFilePath(uid), client)
}

// sendHTTPProfileRequest sends the http request to the given url,
// writes the response down to the given output and returns the profiling run instance.
func sendHTTPProfileRequest(rtype runs.RunType, method, url, token, outputPath string, client *http.Client) runs.ProfilingRun {
	run := runs.ProfilingRun{
		Type:      rtype,
		BeginTime: time.Now(),
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("failed to create http request: %v", err)
		return run
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	res, err := client.Do(req)
	if err != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("failed sending profiling request: %v", err)
		return run
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error status code received: %d", res.StatusCode)
		return run
	}

	if err := writeToFile(res.Body, outputPath); err != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("failed writing profiling data into file: %v", err)
		return run
	}

	run.EndTime = time.Now()
	run.Successful = true

	return run
}

type httpTransportBuilder struct {
	tlsClientConfig *tls.Config
	dialContext     func(ctx context.Context, network, addr string) (net.Conn, error)
}

func newDefaultHTTPTransport() *httpTransportBuilder {
	return &httpTransportBuilder{
		dialContext: (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	}
}

func (b *httpTransportBuilder) withRootCAs(certs *x509.CertPool) *httpTransportBuilder {
	b.tlsClientConfig = &tls.Config{RootCAs: certs, MinVersion: tls.VersionTLS12}
	return b
}

func (b *httpTransportBuilder) withUnixDialContext(socket string) *httpTransportBuilder {
	b.dialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext(ctx, "unix", socket)
	}
	return b
}

func (b *httpTransportBuilder) build() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           b.dialContext,
		TLSClientConfig:       b.tlsClientConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

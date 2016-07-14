// Copyright 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// This file implements an http.Client with request timeouts set by command
// line flags.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/memcache"
)

var (
	dialTimeout    = flag.Duration("dial_timeout", 5*time.Second, "Timeout for dialing an HTTP connection.")
	requestTimeout = flag.Duration("request_timeout", 20*time.Second, "Time out for roundtripping an HTTP request.")
)

type transport struct {
	t http.Transport
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	timer := time.AfterFunc(*requestTimeout, func() {
		t.t.CancelRequest(req)
		log.Printf("Canceled request for %s", req.URL)
	})
	defer timer.Stop()

	req.Header.Set("Authorization", "token "+oAuthToken)
	return t.t.RoundTrip(req)
}

type timeoutConn struct {
	net.Conn
}

func (c timeoutConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.Conn.SetReadDeadline(time.Time{})
	return n, err
}

func timeoutDial(network, addr string) (net.Conn, error) {
	c, err := net.DialTimeout(network, addr, *dialTimeout)
	if err != nil {
		return c, err
	}
	// The net/http transport CancelRequest feature does not work until after
	// the TLS handshake is complete. To help catch hangs during the TLS
	// handshake, we set a deadline on the connection here and clear the
	// deadline when the first read on the connection completes. This is not
	// perfect, but it does catch the case where the server accepts and ignores
	// a connection.
	c.SetDeadline(time.Now().Add(*requestTimeout))
	return timeoutConn{c}, nil
}

func newHTTPClient() *http.Client {

	return &http.Client{Transport: &transport{
		t: http.Transport{
			Dial: timeoutDial,
			ResponseHeaderTimeout: *requestTimeout / 2,
		}}}
}

func newCacheTransport() *httpcache.Transport {
	// host and port are set by GAE Flex runtime, can be left blank locally.
	host := os.Getenv("MEMCACHE_PORT_11211_TCP_ADDR")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MEMCACHE_PORT_11211_TCP_PORT")
	if port == "" {
		port = "11211"
	}
	addr := fmt.Sprintf("%s:%s", host, port)

	return httpcache.NewTransport(
		memcache.New(addr),
	)
}

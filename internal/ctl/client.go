package ctl

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SocketEnv is the variable ym sets for every program it hosts, so `ym ctl`
// run inside a session reaches the ym that started it.
const SocketEnv = "YM_SOCKET"

// FindSocket resolves which running ym to talk to: YM_SOCKET when set,
// otherwise the live sockets in SocketDirs. With several live instances and
// no YM_SOCKET it refuses rather than guess. Dead sockets are removed on the
// way.
func FindSocket() (string, error) {
	if p := os.Getenv(SocketEnv); p != "" {
		return p, nil
	}
	dirs, err := SocketDirs()
	if err != nil {
		return "", err
	}
	var matches []string
	for _, dir := range dirs {
		found, _ := filepath.Glob(filepath.Join(dir, "ym-*.sock"))
		matches = append(matches, found...)
	}
	sort.Strings(matches)
	var live []string
	for _, p := range matches {
		conn, err := net.DialTimeout("unix", p, time.Second)
		if err != nil {
			_ = os.Remove(p) // nobody is listening: a crashed ym left it
			continue
		}
		_ = conn.Close()
		live = append(live, p)
	}
	switch len(live) {
	case 0:
		return "", errors.New("no running ym found (is ym open? set YM_SOCKET to pick one)")
	case 1:
		return live[0], nil
	default:
		return "", fmt.Errorf("several ym instances are running; set %s to one of:\n  %s",
			SocketEnv, strings.Join(live, "\n  "))
	}
}

// Call sends one request to the ym at path and returns its response. A
// response carrying Error is returned as a Go error.
func Call(path string, req Request) (Response, error) {
	conn, err := net.DialTimeout("unix", path, 2*time.Second)
	if err != nil {
		return Response{}, fmt.Errorf("connect to ym: %w", err)
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, err
	}
	_ = conn.SetReadDeadline(time.Now().Add(replyTimeout + 10*time.Second))
	var resp Response
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("read reply from ym: %w", err)
	}
	if !resp.OK {
		if resp.Error == "" {
			resp.Error = "ym refused the request"
		}
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}

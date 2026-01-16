package targets

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

type Target struct {
	Raw         string
	Node        *enode.Node
	NodeString  string
	TCPEndpoint string
	UDPEndpoint string
}

func Parse(input string) (*Target, error) {
	n, err := parseNode(input)
	if err != nil {
		return nil, err
	}

	t := &Target{Raw: input, Node: n, NodeString: n.String()}
	if ep, ok := n.TCPEndpoint(); ok {
		t.TCPEndpoint = ep.String()
	}
	if ep, ok := n.UDPEndpoint(); ok {
		t.UDPEndpoint = ep.String()
	}
	return t, nil
}

func ResolveTCPEndpoint(input string) (string, error) {
	host, portStr, err := net.SplitHostPort(input)
	if err == nil {
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			return "", fmt.Errorf("invalid endpoint %q", input)
		}
		if host == "" {
			return "", fmt.Errorf("invalid endpoint %q", input)
		}
		return net.JoinHostPort(host, strconv.Itoa(port)), nil
	}

	t, err := Parse(input)
	if err != nil {
		return "", err
	}
	if t.TCPEndpoint == "" {
		return "", errors.New("node has no TCP endpoint")
	}
	return t.TCPEndpoint, nil
}

func parseNode(input string) (*enode.Node, error) {
	if strings.HasPrefix(input, "enode://") {
		return enode.ParseV4(input)
	}
	r, err := parseRecord(input)
	if err != nil {
		return nil, err
	}
	return enode.New(enode.ValidSchemes, r)
}

func parseRecord(source string) (*enr.Record, error) {
	bin := []byte(source)
	if d, ok := decodeRecordHex(bytes.TrimSpace(bin)); ok {
		bin = d
	} else if d, ok := decodeRecordBase64(bytes.TrimSpace(bin)); ok {
		bin = d
	}
	var r enr.Record
	if err := rlp.DecodeBytes(bin, &r); err != nil {
		return nil, fmt.Errorf("invalid record %q: %v", source, err)
	}
	return &r, nil
}

func decodeRecordHex(b []byte) ([]byte, bool) {
	if bytes.HasPrefix(b, []byte("0x")) {
		b = b[2:]
	}
	dec := make([]byte, hex.DecodedLen(len(b)))
	_, err := hex.Decode(dec, b)
	return dec, err == nil
}

func decodeRecordBase64(b []byte) ([]byte, bool) {
	if bytes.HasPrefix(b, []byte("enr:")) {
		b = b[4:]
	}
	dec := make([]byte, base64.RawURLEncoding.DecodedLen(len(b)))
	n, err := base64.RawURLEncoding.Decode(dec, b)
	return dec[:n], err == nil
}

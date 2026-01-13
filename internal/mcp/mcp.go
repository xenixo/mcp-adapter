// Package mcp provides MCP protocol utilities.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Message represents a generic MCP JSON-RPC message.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC error.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

// Transport defines the interface for MCP transports.
type Transport interface {
	Send(msg *Message) error
	Receive() (*Message, error)
	Close() error
}

// StdioTransport implements MCP transport over stdio.
type StdioTransport struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

// NewStdioTransport creates a new stdio transport.
func NewStdioTransport(r io.Reader, w io.Writer) *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

// Send sends a message over the transport.
func (t *StdioTransport) Send(msg *Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Write message followed by newline
	if _, err := t.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// Receive receives a message from the transport.
func (t *StdioTransport) Receive() (*Message, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// Close closes the transport.
func (t *StdioTransport) Close() error {
	return nil
}

// Proxy proxies messages between two transports.
type Proxy struct {
	client Transport
	server Transport
}

// NewProxy creates a new message proxy.
func NewProxy(client, server Transport) *Proxy {
	return &Proxy{
		client: client,
		server: server,
	}
}

// Run starts proxying messages bidirectionally.
func (p *Proxy) Run() error {
	errChan := make(chan error, 2)

	// Client -> Server
	go func() {
		for {
			msg, err := p.client.Receive()
			if err != nil {
				errChan <- err
				return
			}
			if err := p.server.Send(msg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Server -> Client
	go func() {
		for {
			msg, err := p.server.Receive()
			if err != nil {
				errChan <- err
				return
			}
			if err := p.client.Send(msg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	return <-errChan
}

package cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// Client is a minimal Redis client implemented with the RESP protocol.
type Client struct {
	addr     string
	password string
}

// Connect initialises a new client and verifies the connection.
func Connect(ctx context.Context, host string, port int, password string) (*Client, error) {
	client := &Client{addr: fmt.Sprintf("%s:%d", host, port), password: password}
	if err := client.ping(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) dial(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	reader := bufio.NewReader(conn)
	if c.password != "" {
		if err := c.send(conn, reader, "AUTH", c.password); err != nil {
			conn.Close()
			return nil, nil, err
		}
	}
	return conn, reader, nil
}

func (c *Client) send(conn net.Conn, reader *bufio.Reader, command string, args ...string) error {
	if err := writeCommand(conn, command, args...); err != nil {
		return err
	}
	resp, err := parseResp(reader)
	if err != nil {
		return err
	}
	if respErr, ok := resp.(respError); ok {
		return errors.New(string(respErr))
	}
	return nil
}

func (c *Client) ping(ctx context.Context) error {
	conn, reader, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := c.send(conn, reader, "PING"); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// Get fetches a string value by key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	conn, reader, err := c.dial(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := writeCommand(conn, "GET", key); err != nil {
		return "", err
	}
	resp, err := parseResp(reader)
	if err != nil {
		return "", err
	}

	switch v := resp.(type) {
	case respBulkString:
		return string(v), nil
	case respNil:
		return "", nil
	case respError:
		return "", errors.New(string(v))
	default:
		return "", fmt.Errorf("unexpected response type %T", resp)
	}
}

// Set saves a key without an expiration.
func (c *Client) Set(ctx context.Context, key, value string) error {
	return c.execSimple(ctx, "SET", key, value)
}

// SetWithExpireAt stores a value with an absolute Unix timestamp expiration.
func (c *Client) SetWithExpireAt(ctx context.Context, key, value string, unixTS int64) error {
	duration := time.Until(time.Unix(unixTS, 0))
	if duration <= 0 {
		return c.Del(ctx, key)
	}
	seconds := int(duration.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return c.execSimple(ctx, "SETEX", key, strconv.Itoa(seconds), value)
}

// Del removes a key from Redis.
func (c *Client) Del(ctx context.Context, key string) error {
	return c.execSimple(ctx, "DEL", key)
}

// Close is a no-op for compatibility.
func (c *Client) Close() error { return nil }

func (c *Client) execSimple(ctx context.Context, command string, args ...string) error {
	conn, reader, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := writeCommand(conn, command, args...); err != nil {
		return err
	}
	resp, err := parseResp(reader)
	if err != nil {
		return err
	}
	if respErr, ok := resp.(respError); ok {
		return errors.New(string(respErr))
	}
	return nil
}

type respValue interface{}

type respError string

type respBulkString []byte

type respNil struct{}

type respInteger int64

func writeCommand(conn net.Conn, command string, args ...string) error {
	var builder strings.Builder
	total := 1 + len(args)
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(total))
	builder.WriteString("\r\n")
	parts := append([]string{command}, args...)
	for _, part := range parts {
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(len(part)))
		builder.WriteString("\r\n")
		builder.WriteString(part)
		builder.WriteString("\r\n")
	}
	_, err := conn.Write([]byte(builder.String()))
	return err
}

func parseResp(reader *bufio.Reader) (respValue, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")

	switch prefix {
	case '+':
		return line, nil
	case '-':
		return respError(line), nil
	case ':':
		n, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return nil, err
		}
		return respInteger(n), nil
	case '$':
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if n == -1 {
			return respNil{}, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return nil, err
		}
		return respBulkString(buf[:n]), nil
	default:
		return nil, fmt.Errorf("unsupported RESP prefix %q", prefix)
	}
}

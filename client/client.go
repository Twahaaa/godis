package client

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/tidwall/resp"
)

type Client struct {
	addr string
	conn net.Conn
}

func New(addr string) *Client {
	conn, err := net.Dial("tcp", addr)

	if err!=nil{
		log.Fatal(err)
	}

	return &Client{
		addr: addr,
		conn: conn,
	}
}

func (c *Client) Set(ctx context.Context, key string, val string) error {
	buf := &bytes.Buffer{}

	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{
		resp.StringValue("SET"),
		resp.StringValue(key),
		resp.StringValue(val),
	})

	_, err := c.conn.Write(buf.Bytes())
	return err
}


func (c *Client) Del(ctx context.Context, key string) (bool, error) {
	buf := &bytes.Buffer{}

	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{
		resp.StringValue("DEL"),
		resp.StringValue(key),
	})

	if _, err := c.conn.Write(buf.Bytes()); err != nil {
		return false, err
	}

	b := make([]byte, 8)
	n, err := c.conn.Read(b)
	if err != nil {
		return false, err
	}
	return string(b[:n]) == ":1\r\n", nil
}

func (c *Client) Keys(ctx context.Context) ([]string, error) {
	buf := &bytes.Buffer{}

	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{
		resp.StringValue("KEYS"),
		resp.StringValue("*"),
	})

	if _, err := c.conn.Write(buf.Bytes()); err != nil {
		return nil, err
	}

	b := make([]byte, 4096)
	n, err := c.conn.Read(b)
	if err != nil {
		return nil, err
	}

	response := string(b[:n])
	if strings.HasPrefix(response, "-ERR") {
		return nil, fmt.Errorf("%s", strings.TrimSpace(response[4:]))
	}
	return strings.Split(response, ","), nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	buf := &bytes.Buffer{}

	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{
		resp.StringValue("GET"),
		resp.StringValue(key),
	})

	_, err := c.conn.Write(buf.Bytes())

	if err!=nil{
		return "", err
	}

	b := make([]byte, 1024)
	n, err := c.conn.Read(b)
	if err != nil {
		return "", err
	}

	response := string(b[:n])
	if strings.HasPrefix(response, "-ERR") {
		return "", fmt.Errorf("%s", strings.TrimSpace(response[4:]))
	}
	return response, nil
}

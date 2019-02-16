package memcache

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
)

var (
	crlf         = []byte("\r\n")
	space        = []byte(" ")
	resultStored = []byte("STORED\r\n")
	resultEnd    = []byte("END\r\n")
)

// conn is a connection to a server.
type conn struct {
	nc     net.Conn
	rw     *bufio.ReadWriter
	addr   net.Addr
	Client *Client
}

// Client is a memcache client.
type Client struct {
	nc net.Conn
	rw *bufio.ReadWriter
	//Timeout      time.Duration
	//MaxIdleConns int
	//lk           sync.Mutex
	//freeconn     map[string][]*conn
}

// Item is an item to be got or stored in a memcached server.
type Item struct {
	Key        string
	Value      []byte
	Flags      uint32
	Expiration int32
	casid      uint64
}

func GetClient(host string) (*Client, error) {
	nc, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	return &Client{
		nc: nc,
		rw: bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
	}, nil
}

func (c *Client) Action(act string, storeItem *Item, key string) (getItem *Item, err error) {
	switch act {
	case "add", "set", "replace", "cas":
		return nil, c.actionCommon(c.rw, act, storeItem)
	case "get":
		actionGet(c.rw, []string{key}, func(item *Item) {
			getItem = item
		})
		return getItem, nil
	}
	return nil, nil
}

func actionGet(rw *bufio.ReadWriter, keys []string, cb func(*Item)) error {
	if _, err := fmt.Fprintf(rw, "gets %s\r\n", strings.Join(keys, " ")); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}
	if err := actionGetResponse(rw.Reader, cb); err != nil {
		return err
	}
	return nil
}

func actionGetResponse(r *bufio.Reader, cb func(*Item)) error {
	for {
		//oneline is end with \n\r
		line, err := r.ReadSlice('\n')
		if err != nil {
			return err
		}
		//read end and return
		if bytes.Equal(line, resultEnd) {
			return nil
		}
		//the line value is the store value info
		it := new(Item)
		size, err := parseGetResponse(line, it)
		if err != nil {
			return err
		}
		//read all value with io and two more byte for \n\r
		it.Value = make([]byte, size+2)
		_, err = io.ReadFull(r, it.Value)
		if err != nil {
			it.Value = nil
			return err
		}
		//if last two bytes is not \r\n,the value is error
		if !bytes.HasSuffix(it.Value, crlf) {
			it.Value = nil
			return fmt.Errorf("memcache: read error with \\r\\n")
		}
		it.Value = it.Value[:size]
		//return value if has read value
		cb(it)
	}
}

func parseGetResponse(line []byte, it *Item) (count int, err error) {
	rule := "VALUE %s %d %d %d\r\n"
	//gets command will return cas ID
	dest := []interface{}{&it.Key, &it.Flags, &count, &it.casid}
	if bytes.Count(line, space) == 3 {
		rule = "VALUE %s %d %d\r\n"
		dest = dest[:3]
	}
	n, err := fmt.Sscanf(string(line), rule, dest...)
	if err != nil || n != len(dest) {
		return -1, fmt.Errorf("memcache: error(%q) with response: %q", err, line)
	}
	return count, nil
}

//actionCommon common action for set add replace add cas
func (c *Client) actionCommon(rw *bufio.ReadWriter, act string, item *Item) error {
	var err error
	if act == "cas" {
		_, err = fmt.Fprintf(rw, "%s %s %d %d %d %d\r\n",
			act, item.Key, item.Flags, item.Expiration, len(item.Value), item.casid)
	} else {
		_, err = fmt.Fprintf(rw, "%s %s %d %d %d\r\n",
			act, item.Key, item.Flags, item.Expiration, len(item.Value))
	}
	if err != nil {
		return err
	}
	if _, err = rw.Write(item.Value); err != nil {
		return err
	}
	if _, err := rw.Write(crlf); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}
	line, err := rw.ReadSlice('\n')
	if err != nil {
		return err
	}
	switch {
	case bytes.Equal(line, resultStored):
		return nil
	default:
		return fmt.Errorf(string(line))
	}
}

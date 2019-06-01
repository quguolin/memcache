package memcache

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

var (
	ErrNotStored   = errors.New("memcache: item not stored")
	ErrCASConflict = errors.New("memcache: compare-and-swap conflict")
	ErrCacheMiss   = errors.New("memcache: cache miss")
)
var (
	crlf            = []byte("\r\n")
	space           = []byte(" ")
	resultStored    = []byte("STORED\r\n")
	resultNotStored = []byte("NOT_STORED\r\n")
	resultExists    = []byte("EXISTS\r\n")
	resultOK        = []byte("OK\r\n")
	resultNotFound  = []byte("NOT_FOUND\r\n")
	resultEnd       = []byte("END\r\n")
	resultDeleted   = []byte("DELETED\r\n")
)

const (
	//FlagRaw default flag byte value
	FlagRaw = 0
	//FlagJson json value
	FlagJson = 1
)

// Connect is a memcache client.
type Connect struct {
	nc net.Conn
	rw *bufio.ReadWriter
	//ebuf json encode buf
	ebuf bytes.Buffer
	//je json encoder
	je *json.Encoder
	//jd json decoder
	jd *json.Decoder
	//jr bytes reader
	jr  bytes.Reader
	mux sync.Mutex
}

func New(host string) (*Connect, error) {
	nc, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	c := &Connect{
		nc: nc,
		rw: bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
	}
	c.je = json.NewEncoder(&c.ebuf)
	c.jd = json.NewDecoder(&c.jr)
	return c, nil
}

//Set set action
func (c *Connect) Set(storeItem *Item) (err error) {
	return c.actionCommon(c.rw, "set", storeItem)
}

//Add add action
func (c *Connect) Add(storeItem *Item) (err error) {
	return c.actionCommon(c.rw, "add", storeItem)
}

//Replace replace actioin
func (c *Connect) Replace(storeItem *Item) (err error) {
	return c.actionCommon(c.rw, "replace", storeItem)
}

//Cas cas action
func (c *Connect) Cas(storeItem *Item) (err error) {
	return c.actionCommon(c.rw, "cas", storeItem)
}

//Get get action
func (c *Connect) Get(key string) (i *Item, err error) {
	defer c.Close()
	err = actionGet(c.rw, []string{key}, func(item *Item) {
		i = item
	})
	return
}

//Scan get item
func (c *Connect) Scan(item *Item, v interface{}) (err error) {
	if err = c.decode(item, v); err != nil {
		return
	}
	return
}

func (c *Connect) Close() error {
	c.mux.Lock()
	err := c.nc.Close()
	c.mux.Unlock()
	return err
}

//Delete delete action
func (c *Connect) Delete(key string) error {
	line, err := writeReadLine(c.rw, "delete %s\r\n", key)
	if err != nil {
		return err
	}
	if bytes.Equal(line, resultDeleted) {
		return nil
	}
	return fmt.Errorf(string(line))
}

//Flush flush all action
func (c *Connect) Flush() error {
	line, err := writeReadLine(c.rw, "flush_all \r\n")
	if err != nil {
		return err
	}
	if bytes.Equal(line, resultOK) {
		return nil
	}
	return fmt.Errorf(string(line))
}

func writeReadLine(rw *bufio.ReadWriter, format string, args ...interface{}) ([]byte, error) {
	_, err := fmt.Fprintf(rw, format, args...)
	if err != nil {
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		return nil, err
	}
	line, err := rw.ReadSlice('\n')
	return line, err
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
	dest := []interface{}{&it.Key, &it.Flags, &count, &it.Casid}
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

func (c *Connect) encode(item *Item) (value []byte, err error) {
	switch item.Flags {
	case FlagRaw:
		value = item.Value
	case FlagJson:
		c.ebuf.Reset()
		if err = c.je.Encode(item.Object); err != nil {
			return
		}
		value = c.ebuf.Bytes()
	default:
		value = item.Value
	}
	return value, nil
}

func (c *Connect) decode(item *Item, v interface{}) (err error) {
	switch item.Flags {
	case FlagJson:
		c.jr.Reset(item.Value)
		err = c.jd.Decode(v)
	default:
		switch v.(type) {
		case *[]byte:

		}
	}
	return
}

//actionCommon common action for set add replace add cas
func (c *Connect) actionCommon(rw *bufio.ReadWriter, act string, item *Item) (err error) {
	var (
		value []byte
	)
	if value, err = c.encode(item); err != nil {
		return
	}
	item.Value = value
	if act == "cas" {
		_, err = fmt.Fprintf(rw, "%s %s %d %d %d %d\r\n",
			act, item.Key, item.Flags, item.Expiration, len(item.Value), item.Casid)
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
	case bytes.Equal(line, resultNotStored):
		return ErrNotStored
	case bytes.Equal(line, resultExists):
		return ErrCASConflict
	case bytes.Equal(line, resultNotFound):
		return ErrCacheMiss
	}
	return fmt.Errorf("memcache: unexpected response line: %q", string(line))
}

package memcache

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotStored   = errors.New("memcache: item not stored")
	ErrNotFound    = errors.New("memcache: item not found")
	ErrCASConflict = errors.New("memcache: compare-and-swap conflict")
	ErrCacheMiss   = errors.New("memcache: cache miss")
	ErrInvalidKey  = errors.New("memcache: key is too long or contains invalid characters")
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
	//_oneChunk 1MB
	_oneChunk = 1000 * 1000 // 1MB
	//_flagMuchChunk value more 1MB flag
	_flagMuchChunk = uint32(1) << 30
)

// Connect is a memcache client.
type Connect struct {
	nc           net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	rw           *bufio.ReadWriter
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

type Config struct {
	host         string
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func New(config *Config) (*Connect, error) {
	nc, err := net.Dial("tcp", config.host)
	if err != nil {
		return nil, err
	}
	c := &Connect{
		nc:           nc,
		rw:           bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
		readTimeout:  config.readTimeout,
		writeTimeout: config.writeTimeout,
	}
	c.je = json.NewEncoder(&c.ebuf)
	c.jd = json.NewDecoder(&c.jr)
	return c, nil
}

//validate key
func valKey(key string) bool {
	if len(key) > 250 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] <= ' ' || key[i] == 0x7f {
			return false
		}
	}
	return true
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
func (c *Connect) Get(key string) (item *Item, err error) {
	if !valKey(key) {
		return nil, ErrInvalidKey
	}
	err = c.actionGet(c.rw, []string{key}, func(i *Item) {
		item = i
	})
	if err != nil {
		return
	}
	if item == nil {
		return nil, ErrNotFound
	}
	if (item.Flags & _flagMuchChunk) != _flagMuchChunk {
		return
	}
	length, err := strconv.Atoi(string(item.Value))
	if err != nil {
		return
	}
	loop := length/_oneChunk + 1
	var (
		keys  []string
		items map[string]*Item
	)
	for i := 1; i <= loop; i++ {
		keys = append(keys, fmt.Sprintf("%s%d", key, i))
	}
	if items, err = c.GetMulti(keys...); err != nil {
		return
	}
	if len(items) != loop {
		return nil, ErrNotFound
	}
	item.Value = make([]byte, 0, length)
	for _, key := range keys {
		v, ok := items[key]
		if !ok || v == nil {
			return nil, ErrNotFound
		}
		item.Value = append(item.Value, v.Value...)
	}
	item.Flags = item.Flags ^ _flagMuchChunk
	return
}

//GetMulti get muti values
func (c *Connect) GetMulti(keys ...string) (items map[string]*Item, err error) {
	for _, key := range keys {
		if !valKey(key) {
			return items, ErrInvalidKey
		}
	}
	items = make(map[string]*Item, len(keys))
	err = c.actionGet(c.rw, keys, func(item *Item) {
		items[item.Key] = item
	})
	if err != nil {
		return
	}
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
	if !valKey(key) {
		return ErrInvalidKey
	}
	line, err := c.writeReadLine(c.rw, "delete %s\r\n", key)
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
	line, err := c.writeReadLine(c.rw, "flush_all \r\n")
	if err != nil {
		return err
	}
	if bytes.Equal(line, resultOK) {
		return nil
	}
	return fmt.Errorf(string(line))
}

func (c *Connect) writeReadLine(rw *bufio.ReadWriter, format string, args ...interface{}) ([]byte, error) {
	c.setWTimeout()
	_, err := fmt.Fprintf(rw, format, args...)
	if err != nil {
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		return nil, err
	}
	c.setRTimeout()
	line, err := rw.ReadSlice('\n')
	return line, err
}

func (c *Connect) actionGet(rw *bufio.ReadWriter, keys []string, cb func(*Item)) error {
	c.setWTimeout()
	if _, err := fmt.Fprintf(rw, "gets %s\r\n", strings.Join(keys, " ")); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}
	if err := c.actionGetResponse(rw.Reader, cb); err != nil {
		return err
	}
	return nil
}

func (c *Connect) actionGetResponse(r *bufio.Reader, cb func(*Item)) error {
	c.setRTimeout()
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

//setRTimeout set net read timeout
func (c *Connect) setRTimeout() {
	if c.readTimeout != 0 {
		c.nc.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
}

//setWTimeout set net write timeout
func (c *Connect) setWTimeout() {
	if c.writeTimeout != 0 {
		c.nc.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}
}

//actionCommon common action for set add replace add cas
func (c *Connect) actionCommon(rw *bufio.ReadWriter, act string, item *Item) (err error) {
	var (
		value []byte
	)
	if !valKey(item.Key) {
		return ErrInvalidKey
	}
	if value, err = c.encode(item); err != nil {
		return
	}
	//one chunk
	if len(value) < _oneChunk {
		return c.actionCommonChunk(rw, act, item, value)
	}
	length := len(value)
	item.Flags = item.Flags | _flagMuchChunk
	if err = c.actionCommonChunk(rw, act, item, []byte(strconv.Itoa(length))); err != nil {
		return
	}
	loop := length/_oneChunk + 1
	var oneChunk []byte
	key := item.Key
	for i := 1; i <= loop; i++ {
		if i == loop {
			oneChunk = value[_oneChunk*(i-1):]
		} else {
			oneChunk = value[_oneChunk*(i-1) : _oneChunk*i]
		}
		item.Key = fmt.Sprintf("%s%d", key, i)
		if err = c.actionCommonChunk(rw, act, item, oneChunk); err != nil {
			return
		}
	}
	return
}

//actionCommon common action for set add replace add cas
func (c *Connect) actionCommonChunk(rw *bufio.ReadWriter, act string, item *Item, chunk []byte) (err error) {
	item.Value = chunk
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
	c.setWTimeout()
	if _, err = rw.Write(item.Value); err != nil {
		return err
	}
	if _, err := rw.Write(crlf); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}
	c.setRTimeout()
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

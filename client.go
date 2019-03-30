package memcache

type Client struct {
	conn MemcacheProtocal
}

// Result is the result of Get
type Result struct {
	item *Item
	conn MemcacheProtocal
	err  error
}

func NewClient(host string) *Client {
	conn, err := New(host)
	if err != nil {
		panic(err)
	}
	return &Client{
		conn: conn,
	}
}
func (mem *Client) Get(key string) *Result {
	conn := mem.conn
	item, err := conn.Get(key)
	if err != nil {
		conn.Close()
	}
	return &Result{
		item: item,
		conn: conn,
		err:  err,
	}
}

func (r *Result) Scan(v interface{}) (err error) {
	if r.err != nil {
		return r.err
	}
	c := r.conn
	defer c.Close()
	err = r.conn.Scan(r.item, v)
	return
}

func (mem *Client) Add(item *Item) (err error) {
	c := mem.conn
	defer c.Close()
	return c.Add(item)
}

func (mem *Client) Set(item *Item) (err error) {
	c := mem.conn
	defer c.Close()
	return c.Set(item)
}

func (mem *Client) Replace(item *Item) (err error) {
	c := mem.conn
	defer c.Close()
	return c.Replace(item)
}

func (mem *Client) Delete(key string) (err error) {
	c := mem.conn
	defer c.Close()
	return c.Delete(key)
}

func (mem *Client) Close() (err error) {
	return mem.conn.Close()
}

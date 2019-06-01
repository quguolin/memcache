package memcache

// Item is an item to be got or stored in a memcached server.
type Item struct {
	//Key key
	Key string
	//Value binary value
	Value []byte
	//Object encode value
	Object interface{}
	//Flags item's type
	Flags uint32
	//Expiration value expiration
	Expiration int32
	//Casid Casid
	Casid uint64
}

//Memcache interface for memcache proto
type MemcacheProtocal interface {
	Get(key string) (i *Item, err error)
	Set(storeItem *Item) (err error)
	Add(storeItem *Item) (err error)
	Replace(storeItem *Item) (err error)
	Scan(item *Item, v interface{}) (err error)
	Cas(storeItem *Item) (err error)
	Delete(key string) error
	Close() error
}

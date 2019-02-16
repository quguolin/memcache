package main

import (
	"fmt"
	"library/memcache"
)

func main() {
	c, err := memcache.GetClient("127.0.0.1:11211")
	if err != nil {
		panic(err)
	}
	i := &memcache.Item{
		Key:   "userId",
		Value: []byte("aaa"),
	}
	item, err := c.Action("set", i, "userId")
	if err != nil {
		panic(err)
	}
	item, err = c.Action("get", nil, "userId")
	if err != nil {
		panic(err)
	}
	fmt.Println(i.Value)
	fmt.Println(string(item.Value))
}

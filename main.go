package main

import (
	"fmt"
	"memcache/memcache"
)

type User struct {
	Name      string
	IsAdmin   bool
	Followers uint
}

func main() {
	c, err := memcache.GetClient("127.0.0.1:11211")
	if err != nil {
		panic(err)
	}
	//user := User{
	//	Name:      "cizixs",
	//	IsAdmin:   true,
	//	Followers: 36,
	//}
	i := &memcache.Item{}
	i = &memcache.Item{
		Key:        "key",
		Flags:      memcache.FlagRaw,
		Expiration: 1000,
		Value:      []byte("value"),
	}
	if err = c.Set(i); err != nil {
		panic(err)
	}
	if i, err = c.Get("key"); err != nil {
		panic(err)
	}
	i.Value = []byte("value new")
	if err = c.Cas(i); err != nil {
		panic(err)
	}
	if i, err = c.Get("key"); err != nil {
		panic(err)
	}
	fmt.Println(string(i.Value))
	fmt.Println("success!")
	//i := &memcache.Item{
	//	Key:    "userId2",
	//	Object: user,
	//	Flags:  memcache.FlagJson,
	//}
	//err = c.Set(i)
	//if err != nil {
	//	panic(err)
	//}
	//i := &memcache.Item{
	//	Key:   "userId",
	//	Value: []byte("aaa"),
	//}
	//err = c.Add(i)
	//if err != nil {
	//	panic(err)
	//}
	//i := &memcache.Item{
	//	Key:   "userId",
	//	Value: []byte("aaa111bbb"),
	//}
	//err = c.Replace(i)
	//if err != nil {
	//	panic(err)
	//}
	//err = c.Delete("userId1")
	//if err != nil {
	//	panic(err)
	//}
	//i, err = c.Get("userId2")
	//if err != nil {
	//	panic(err)
	//}
	//u := &User{}
	//c.Scan(i, u)
	//fmt.Println(u)
}

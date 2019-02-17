package main

import (
	"library/memcache"
)

func main() {
	c, err := memcache.GetClient("127.0.0.1:11211")
	if err != nil {
		panic(err)
	}
	i := &memcache.Item{
		Key:   "userId2",
		Value: []byte("aaa2"),
	}
	err = c.Set(i)
	if err != nil {
		panic(err)
	}
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
	err = c.Delete("userId1")
	if err != nil {
		panic(err)
	}
	//i, err = c.Get("userId2")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(string(i.Value))
}

package memcache

import (
	"fmt"
	"testing"
)

type student struct {
	Name   string
	Gender string
	Age    int
}

func TestClient_Get(t *testing.T) {
	c := NewClient(host)
	stu := &student{}
	if err := c.Get("test").Scan(stu); err != nil {
		panic(err)
	}
	fmt.Println(stu)
}

func TestClient_Add(t *testing.T) {
	s := student{
		Name:   "Moor",
		Gender: "boy",
		Age:    20,
	}
	c := NewClient(host)
	item := &Item{
		Key:        "test",
		Object:     s,
		Flags:      FlagJson,
		Expiration: expire,
	}
	if err := c.Set(item); err != nil {
		panic(err)
	}
}

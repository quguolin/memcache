package memcache

import (
	"bytes"
	"testing"
	"time"
)

const (
	host = "127.0.0.1:11211"
	rTimeout = time.Duration(5*time.Millisecond)
	wTimeout = time.Duration(5*time.Millisecond)
)

var (
	key          = "city"
	value        = "shanghai"
	key2         = "city5"
	value2       = "beijing"
	expire int32 = 100
)

func NewConnect() *Connect {
	c, err := New(&Config{
		host:host,
		readTimeout:rTimeout,
		writeTimeout:wTimeout,
	})
	if err != nil {
		panic(err)
	}
	return c
}

func TestConnect_Scan(t *testing.T) {
	type student struct {
		Name   string
		Gender string
		Age    int
	}
	s := student{
		Name:   "Moor",
		Gender: "boy",
		Age:    20,
	}
	c := NewConnect()
	defer c.Close()
	item := &Item{
		Key:        key,
		Object:     s,
		Flags:      FlagJson,
		Expiration: expire,
	}
	err := c.Set(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	stu := &student{}
	err = c.Scan(item, stu)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if stu == nil {
		t.Errorf("item is null")
		return
	}
	if stu.Name != s.Name || stu.Gender != s.Gender || stu.Age != s.Age {
		t.Errorf("item is equal set item")
		return
	}
}

func TestConnect_Flush(t *testing.T) {
	c := NewConnect()
	defer c.Close()
	item := &Item{
		Key:        key,
		Value:      []byte(value),
		Flags:      FlagRaw,
		Expiration: expire,
	}
	err := c.Set(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = c.Flush()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if item != nil {
		t.Errorf("item is not flush")
		return
	}
}

func TestConnect_Delete(t *testing.T) {
	c := NewConnect()
	defer c.Close()
	item := &Item{
		Key:        key,
		Value:      []byte(value),
		Flags:      FlagRaw,
		Expiration: expire,
	}
	err := c.Set(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	err = c.Delete(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if item != nil {
		t.Errorf("item is not deleted")
		return
	}
}

func TestConnect_Cas(t *testing.T) {
	c := NewConnect()
	//defer c.Close()
	item := &Item{
		Key:        key,
		Value:      []byte(value),
		Flags:      FlagRaw,
		Expiration: expire,
	}
	err := c.Set(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if item == nil {
		t.Errorf("get null item")
		return
	}
	if item.Casid == 0 {
		t.Errorf("get item's casid is null")
		return
	}
	item.Value = []byte(value2)
	err = c.Cas(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if !bytes.Equal(item.Value, []byte(value2)) {
		t.Errorf("item value is not equal set")
		return
	}
}

func TestConnect_Get(t *testing.T) {
	c := NewConnect()
	defer c.Close()
	s := student{
		Name:   "Moor",
		Gender: "boy",
		Age:    20,
	}
	item := &Item{
		Key:        key,
		Object:     s,
		Flags:      FlagJson,
		Expiration: expire,
	}
	err := c.Set(item)

	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if item == nil {
		t.Errorf("get null item")
		return
	}
	t.Log("pass")
}

func TestConnect_Replace(t *testing.T) {
	c := NewConnect()
	defer c.Close()
	item := &Item{
		Key:        key,
		Value:      []byte(value),
		Flags:      FlagRaw,
		Expiration: expire,
	}
	err := c.Replace(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if item == nil {
		t.Errorf("get null item")
		return
	}
	if !bytes.Equal(item.Value, []byte(value)) {
		t.Errorf("get item error")
		return
	}
	t.Log("pass")
}


func TestConnect_Add(t *testing.T) {
	c := NewConnect()
	defer c.Close()
	err := c.Flush()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item := &Item{
		Key:        key,
		Value:      []byte(value),
		Flags:      FlagRaw,
		Expiration: expire,
	}
	err = c.Add(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if item == nil {
		t.Errorf("get null item")
		return
	}
	if !bytes.Equal(item.Value, []byte(value)) {
		t.Errorf("get item error")
		return
	}
	t.Log("pass")
}

func TestConnect_Set(t *testing.T) {
	c := NewConnect()
	defer c.Close()
	item := &Item{
		Key:        key,
		Value:      []byte(value),
		Flags:      FlagRaw,
		Expiration: expire,
	}
	err := c.Set(item)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	item, err = c.Get(key)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if item == nil {
		t.Errorf("get null item")
		return
	}
	if !bytes.Equal(item.Value, []byte(value)) {
		t.Errorf("get item error")
		return
	}
	t.Log("pass")
}

package main

import "sync"

type KV struct{
	mu sync.RWMutex
	data map[string][]byte
}

func NewKV() *KV{
	return &KV{
		data: map[string][]byte{},
	}
}

func (kv *KV) Set(key string, val string) error{
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.data[key] = 	[]byte(val)
	return nil
} 
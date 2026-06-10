package main

import (
	"maps"
	"slices"
	"sync"
)

type KV struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewKV() *KV {
	return &KV{
		data: map[string][]byte{},
	}
}

func (kv *KV) Set(key []byte, val []byte) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.data[string(key)] = []byte(val)
	return nil
}

func (kv *KV) Get(key []byte) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok := kv.data[string(key)]
	return val, ok
}

func (kv *KV) Del(key []byte) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	_, ok := kv.data[string(key)]
	delete(kv.data, string(key))
	return ok
}

func (kv *KV) Exists(key []byte) bool {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	_, ok := kv.data[string(key)]
	return ok
}

func (kv *KV) Keys() []string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	keys := slices.Collect(maps.Keys(kv.data))
	return keys
}

package main

import (
	"sync"
	"time"
)

type KV struct {
	mu      sync.RWMutex
	data    map[string][]byte
	expires map[string]time.Time
}

func NewKV() *KV {
	kv := &KV{
		data:    map[string][]byte{},
		expires: map[string]time.Time{},
	}

	go func() {
		for range time.Tick(time.Second * 5) {
			kv.mu.Lock()
			for key, value := range kv.expires {
				if time.Now().After(value) {
					delete(kv.expires, key)
					delete(kv.data, key)
				}
			}
			kv.mu.Unlock()
		}
	}()

	return kv
}

func (kv *KV) Set(key []byte, val []byte, ttl time.Duration) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.data[string(key)] = []byte(val)
	if ttl > 0 {
		kv.expires[string(key)] = time.Now().Add(ttl)
	} else {
		delete(kv.expires, string(key))
	}
	return nil
}

func (kv *KV) Get(key []byte) ([]byte, bool) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if exp, ok := kv.expires[string(key)]; ok {
		if time.Now().After(exp) {
			delete(kv.data, string(key))
			delete(kv.expires, string(key))
			return nil, false
		}
	}
	val, ok := kv.data[string(key)]
	return val, ok
}

func (kv *KV) Del(key []byte) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	_, ok := kv.data[string(key)]
	delete(kv.data, string(key))
	delete(kv.expires, string(key))
	return ok
}

func (kv *KV) Exists(key []byte) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if exp, ok := kv.expires[string(key)]; ok {
		if time.Now().After(exp) {
			delete(kv.data, string(key))
			delete(kv.expires, string(key))
			return false
		}
	}
	_, ok := kv.data[string(key)]
	return ok
}

func (kv *KV) Keys() []string {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	var keys[] string

	for key := range kv.data{
		if exp, ok := kv.expires[string(key)]; ok{
			if time.Now().After(exp){
				delete(kv.data, string(key))
				delete(kv.expires, string(key))
				continue
			}
		}
		keys = append(keys, key)
	}
	return keys
}

func (kv *KV) TTL(key []byte) (time.Duration, int) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if exp, ok := kv.expires[string(key)]; ok{
		if time.Now().After(exp){
			delete(kv.data, string(key))
			delete(kv.expires, string(key))
		}
	}

	_, ok := kv.data[string(key)]

	if !ok{
		return 0 ,-2
	}

	val, ok := kv.expires[string(key)]

	if !ok{
		return 0,-1
	}

	return time.Until(val), 0
}

func (kv *KV) Expire(key []byte,ttl time.Duration) bool{
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if exp, ok := kv.expires[string(key)]; ok{
		if time.Now().After(exp){
			delete(kv.data, string(key))
			delete(kv.expires, string(key))
		}
	}

	_, ok := kv.data[string(key)]

	if !ok{
		return false
	}

	kv.expires[string(key)] = time.Now().Add(ttl)

	return true
}
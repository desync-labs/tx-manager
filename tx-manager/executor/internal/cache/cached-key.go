package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

type KeyCacheInterface interface {
	Set(publicKey, privateKey string)
	Get(publicKey string) (string, error)
}

type KeyCache struct {
	mu        sync.RWMutex
	cache     map[string]cachedKey
	ttl       time.Duration
	cleanupCh chan struct{}
	client    *api.Client
}

type cachedKey struct {
	privateKey string
	expiresAt  time.Time
}

func NewKeyCache(client *api.Client, ttl time.Duration) *KeyCache {
	c := &KeyCache{
		cache:     make(map[string]cachedKey),
		ttl:       ttl,
		cleanupCh: make(chan struct{}),
		client:    client,
	}
	go c.cleanupRoutine()
	return c
}

func (c *KeyCache) Set(publicKey, privateKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[publicKey] = cachedKey{
		privateKey: privateKey,
		expiresAt:  time.Now().Add(c.ttl),
	}
}

func (kc *KeyCache) Get(publicKey string) (string, error) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	key, exists := kc.cache[publicKey]
	if !exists || time.Now().After(key.expiresAt) {

		slog.Debug("Cache miss", "public-key", publicKey)

		// Fetch from Vault
		secret, err := kc.client.KVv1("private-keys").Get(context.Background(), publicKey)
		if err != nil {
			return "", fmt.Errorf("failed to get private key from Vault: %v", err)
		}

		privateKey, ok := secret.Data["private_key"].(string)
		if !ok {
			return "", fmt.Errorf("invalid private_key format for public key: %s", publicKey)
		}

		// Cache the key
		kc.cache[publicKey] = cachedKey{
			privateKey: privateKey,
			expiresAt:  time.Now().Add(kc.ttl),
		}

		return privateKey, nil
	}

	return key.privateKey, nil
}

func (c *KeyCache) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.cleanupCh:
			return
		}
	}
}

func (c *KeyCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.cache {
		if now.After(v.expiresAt) {
			delete(c.cache, k)
		}
	}
}

func (c *KeyCache) Close() {
	close(c.cleanupCh)
}

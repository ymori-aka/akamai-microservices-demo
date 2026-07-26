// Copyright 2026 Akamai
//
// Licensed under the Apache License, Version 2.0 (the "License").

// Best-seller ranking, backed by a Valkey/Redis Sorted Set.
//
// On each placed order we ZINCRBY the ordered quantity into the
// "sales:ranking:units" sorted set, keyed by product id. The frontend
// reads the top members back with ZREVRANGE to render a live
// best-sellers ranking. This showcases Valkey's sorted-set as an O(log N)
// atomic leaderboard.
//
// Activated only when REDIS_URL is set (a rediss:// URL for Akamai
// Managed Valkey). Otherwise ranking updates are silently skipped, so
// the service keeps working exactly as before when Valkey is absent.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"sync"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
	"github.com/redis/go-redis/v9"
)

// RankingKey is the sorted set holding units-sold per product id. Shared
// with the frontend reader (keep in sync).
const RankingKey = "sales:ranking:units"

var (
	rankingRDB  *redis.Client
	rankingOnce sync.Once
)

// initRanking connects to Valkey from REDIS_URL (rediss://...). When
// REDIS_CA_CERT_PATH is set, the mounted CA is used to verify Valkey's
// TLS chain — the runtime image is distroless (no shell), so the cert
// can't be installed into the OS trust store; we pin it in-process.
// Returns nil and leaves rankingRDB == nil when REDIS_URL is absent.
func initRanking() error {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		log.Info("REDIS_URL not set; best-seller ranking disabled.")
		return nil
	}

	var initErr error
	rankingOnce.Do(func() {
		opt, err := redis.ParseURL(url)
		if err != nil {
			initErr = err
			return
		}
		if caPath := os.Getenv("REDIS_CA_CERT_PATH"); caPath != "" {
			pem, err := os.ReadFile(caPath)
			if err != nil {
				initErr = err
				return
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				log.Warn("ranking: no certs parsed from REDIS_CA_CERT_PATH")
			}
			if opt.TLSConfig == nil {
				opt.TLSConfig = &tls.Config{}
			}
			opt.TLSConfig.RootCAs = pool
		}
		rankingRDB = redis.NewClient(opt)
		log.Info("best-seller ranking enabled (Valkey sorted set)")
	})
	return initErr
}

// recordSaleRanking bumps the sorted-set score of every ordered product
// by its quantity. Best-effort: any error is logged and swallowed so a
// Valkey hiccup never fails a checkout (mirrors order persistence).
func recordSaleRanking(ctx context.Context, items []*pb.OrderItem) {
	if rankingRDB == nil {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	pipe := rankingRDB.Pipeline()
	for _, it := range items {
		pid := it.GetItem().GetProductId()
		qty := float64(it.GetItem().GetQuantity())
		if pid == "" || qty <= 0 {
			continue
		}
		pipe.ZIncrBy(cctx, RankingKey, qty, pid)
	}
	if _, err := pipe.Exec(cctx); err != nil {
		log.Warnf("ranking: ZINCRBY failed (non-fatal): %v", err)
		return
	}
	log.Debugf("ranking: recorded %d line items", len(items))
}

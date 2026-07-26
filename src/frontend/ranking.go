// Copyright 2026 Akamai
//
// Licensed under the Apache License, Version 2.0 (the "License").

// Read side of the best-seller ranking. checkoutservice writes
// units-sold into a Valkey sorted set on each order (see
// checkoutservice/ranking.go); here the frontend reads the top members
// with ZREVRANGE and resolves them to full products for display.
//
// Enabled only when REDIS_URL is set (a rediss:// URL for Akamai Managed
// Valkey). When absent, getTopSellers returns an empty slice and the UI
// simply omits the ranking — the store keeps working exactly as before.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"sync"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
	"github.com/redis/go-redis/v9"
)

// rankingKey must match checkoutservice/ranking.go's RankingKey.
const rankingKey = "sales:ranking:units"

var (
	rankingRDB  *redis.Client
	rankingOnce sync.Once
)

// RankingView is one row of the best-seller list, resolved to a product.
type RankingView struct {
	Rank  int
	Item  *pb.Product
	Units int64
	Price *pb.Money
}

// initRanking connects to Valkey from REDIS_URL. When REDIS_CA_CERT_PATH
// is set, that CA is pinned in-process for TLS verification (distroless
// runtime image has no OS trust store to install it into).
func initRanking() {
	rankingOnce.Do(func() {
		url := os.Getenv("REDIS_URL")
		if url == "" {
			log.Info("REDIS_URL not set; best-seller ranking disabled.")
			return
		}
		opt, err := redis.ParseURL(url)
		if err != nil {
			log.Warnf("ranking: bad REDIS_URL, disabling ranking: %v", err)
			return
		}
		if caPath := os.Getenv("REDIS_CA_CERT_PATH"); caPath != "" {
			if pem, err := os.ReadFile(caPath); err == nil {
				pool := x509.NewCertPool()
				pool.AppendCertsFromPEM(pem)
				if opt.TLSConfig == nil {
					opt.TLSConfig = &tls.Config{}
				}
				opt.TLSConfig.RootCAs = pool
			} else {
				log.Warnf("ranking: could not read CA cert: %v", err)
			}
		}
		rankingRDB = redis.NewClient(opt)
		log.Info("best-seller ranking enabled (Valkey sorted set)")
	})
}

// rankingEnabled reports whether a Valkey ranking connection is live.
func rankingEnabled() bool { return rankingRDB != nil }

// getTopSellers returns up to n best-selling products (highest units
// first), each resolved to a full product with a localized price.
// Missing/deleted products are skipped. Returns an empty slice (never an
// error to the caller's page) when ranking is disabled or empty.
func (fe *frontendServer) getTopSellers(ctx context.Context, n int, currency string) ([]RankingView, error) {
	if !rankingEnabled() {
		return nil, nil
	}
	z, err := rankingRDB.ZRevRangeWithScores(ctx, rankingKey, 0, int64(n-1)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]RankingView, 0, len(z))
	rank := 1
	for _, m := range z {
		pid, _ := m.Member.(string)
		if pid == "" {
			continue
		}
		p, err := fe.getProductWithAdminFallback(ctx, pid)
		if err != nil || p == nil {
			continue // product removed since it was ranked; skip
		}
		price, err := fe.convertCurrency(ctx, p.GetPriceUsd(), currency)
		if err != nil {
			price = p.GetPriceUsd()
		}
		out = append(out, RankingView{
			Rank:  rank,
			Item:  p,
			Units: int64(m.Score),
			Price: price,
		})
		rank++
	}
	return out, nil
}

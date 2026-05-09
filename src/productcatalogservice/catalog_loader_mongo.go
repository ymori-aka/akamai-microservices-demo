// Copyright 2026 Akamai
//
// Licensed under the Apache License, Version 2.0 (the "License").

package main

import (
	"bytes"
	"context"
	"os"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	"github.com/golang/protobuf/jsonpb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// productDoc represents the MongoDB document layout for a product.
// Multi-language fields are stored as nested maps so the frontend can
// pick its locale at render time without hitting the gRPC service.
type productDoc struct {
	ID          string            `bson:"_id"`
	Name        map[string]string `bson:"name,omitempty"`
	Description map[string]string `bson:"description,omitempty"`
	Picture     string            `bson:"picture"`
	Price       struct {
		CurrencyCode string `bson:"currencyCode"`
		Units        int64  `bson:"units"`
		Nanos        int32  `bson:"nanos"`
	} `bson:"price"`
	Categories []string  `bson:"categories"`
	Stock      int       `bson:"stock,omitempty"`
	Hidden     bool      `bson:"hidden,omitempty"`
	UpdatedAt  time.Time `bson:"updatedAt,omitempty"`
}

const (
	mongoDBName     = "boutique"
	mongoCollection = "products"
)

// loadCatalogFromMongoDB connects to MongoDB, seeds the catalog from
// products.json if the collection is empty, then loads all products
// (English-default) into the in-memory catalog used by the gRPC server.
func loadCatalogFromMongoDB(catalog *pb.ListProductsResponse) error {
	uri := os.Getenv("MONGODB_URI")
	log.Infof("loading catalog from MongoDB...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Warnf("failed to connect MongoDB: %v", err)
		return err
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err := client.Ping(ctx, nil); err != nil {
		log.Warnf("MongoDB ping failed: %v", err)
		return err
	}

	col := client.Database(mongoDBName).Collection(mongoCollection)

	// Seed from products.json on first run (collection empty).
	count, err := col.CountDocuments(ctx, bson.D{})
	if err != nil {
		log.Warnf("count failed: %v", err)
		return err
	}
	if count == 0 {
		log.Info("MongoDB products collection is empty, seeding from products.json")
		if err := seedFromProductsJSON(ctx, col); err != nil {
			log.Warnf("seed failed: %v", err)
			return err
		}
	}

	// Read all non-hidden products.
	cur, err := col.Find(ctx, bson.M{"hidden": bson.M{"$ne": true}})
	if err != nil {
		log.Warnf("find failed: %v", err)
		return err
	}
	defer cur.Close(ctx)

	catalog.Products = catalog.Products[:0]
	for cur.Next(ctx) {
		var d productDoc
		if err := cur.Decode(&d); err != nil {
			log.Warnf("decode failed: %v", err)
			continue
		}
		// Pick English by default for the gRPC service. Frontend can
		// query Mongo directly to get other locales.
		nameEN := d.Name["en"]
		descEN := d.Description["en"]
		// Fallback to any available language if english missing.
		if nameEN == "" {
			for _, v := range d.Name {
				nameEN = v
				break
			}
		}
		if descEN == "" {
			for _, v := range d.Description {
				descEN = v
				break
			}
		}

		catalog.Products = append(catalog.Products, &pb.Product{
			Id:          d.ID,
			Name:        nameEN,
			Description: descEN,
			Picture:     d.Picture,
			PriceUsd: &pb.Money{
				CurrencyCode: d.Price.CurrencyCode,
				Units:        d.Price.Units,
				Nanos:        d.Price.Nanos,
			},
			Categories: d.Categories,
		})
	}

	log.Infof("loaded %d products from MongoDB", len(catalog.Products))
	return nil
}

// seedFromProductsJSON reads products.json (the legacy bootstrap data)
// and inserts each product into the MongoDB collection. The English
// values become name.en / description.en. Other locales can be filled
// in later via the admin UI.
func seedFromProductsJSON(ctx context.Context, col *mongo.Collection) error {
	raw, err := os.ReadFile("products.json")
	if err != nil {
		return err
	}
	var legacy pb.ListProductsResponse
	if err := jsonpb.Unmarshal(bytes.NewReader(raw), &legacy); err != nil {
		return err
	}

	docs := make([]interface{}, 0, len(legacy.Products))
	for _, p := range legacy.Products {
		d := productDoc{
			ID:          p.Id,
			Name:        map[string]string{"en": p.Name},
			Description: map[string]string{"en": p.Description},
			Picture:     p.Picture,
			Categories:  p.Categories,
			Stock:       100,
			Hidden:      false,
			UpdatedAt:   time.Now().UTC(),
		}
		if p.PriceUsd != nil {
			d.Price.CurrencyCode = p.PriceUsd.CurrencyCode
			d.Price.Units = p.PriceUsd.Units
			d.Price.Nanos = p.PriceUsd.Nanos
		}
		docs = append(docs, d)
	}
	if len(docs) == 0 {
		return nil
	}
	res, err := col.InsertMany(ctx, docs)
	if err != nil {
		return err
	}
	log.Infof("seeded %d products into MongoDB", len(res.InsertedIDs))
	return nil
}

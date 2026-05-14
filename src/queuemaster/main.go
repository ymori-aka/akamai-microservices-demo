// queuemaster — consumes order events from RabbitMQ and simulates
// asynchronous fulfillment processing.
//
// Architecture:
//   checkoutservice -> RabbitMQ exchange "orders" -> queue "order.placed"
//                                                          |
//                                                          v
//                                                   queuemaster (this)
//                                                          |
//                                                          v
//                                          RabbitMQ exchange "orders" -> queue "order.fulfilled"
//
// Demo purpose: shows queue depth growing under load (Grafana) while
// queuemaster steadily drains it. Reconnects with backoff if RabbitMQ
// is unavailable so a pod restart of RabbitMQ doesn't crash this pod.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

const (
	exchangeName    = "orders"
	queuePlaced     = "order.placed"
	queueFulfilled  = "order.fulfilled"
	routePlaced     = "order.placed"
	routeFulfilled  = "order.fulfilled"
	healthPort      = ":8080"
	maxBackoff      = 30 * time.Second
	initialBackoff  = 2 * time.Second
)

var (
	ordersReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "queuemaster_orders_received_total",
		Help: "Total number of order events consumed from RabbitMQ.",
	})
	ordersProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "queuemaster_orders_processed_total",
		Help: "Total number of order events successfully processed and re-published.",
	})
	processingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "queuemaster_processing_duration_seconds",
		Help:    "Time spent processing a single order event.",
		Buckets: []float64{0.05, 0.1, 0.2, 0.5, 1.0},
	})

	// connected: 1 when AMQP connection is healthy, 0 otherwise.
	// atomic for lock-free read from the /healthz handler.
	connected atomic.Int32
)

func init() {
	prometheus.MustRegister(ordersReceived, ordersProcessed, processingDuration)
	log.SetFormatter(&log.JSONFormatter{})
}

type orderEvent struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	TrackingID string `json:"tracking_id"`
	ItemCount  int    `json:"item_count"`
}

func main() {
	rabbitAddr := os.Getenv("RABBITMQ_ADDR")
	if rabbitAddr == "" {
		rabbitAddr = "rabbitmq:5672"
	}

	// HTTP server (health + metrics) runs unconditionally so liveness
	// checks pass even when AMQP is reconnecting.
	go serveHTTP()

	// Graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info("shutdown signal received")
		cancel()
	}()

	// Reconnect loop with exponential backoff.
	backoff := initialBackoff
	for {
		if ctx.Err() != nil {
			log.Info("queuemaster exiting")
			return
		}
		if err := runOnce(ctx, rabbitAddr); err != nil {
			connected.Store(0)
			log.WithError(err).Warnf("AMQP loop exited, reconnecting in %s", backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		// Clean exit (only happens on ctx cancel) — reset backoff.
		backoff = initialBackoff
	}
}

// runOnce establishes a connection, declares topology, and consumes
// until the connection dies or the context is cancelled.
func runOnce(ctx context.Context, addr string) error {
	conn, err := amqp.Dial("amqp://" + addr)
	if err != nil {
		return fmt.Errorf("amqp dial: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("amqp channel: %w", err)
	}
	defer ch.Close()

	if err := declareTopology(ch); err != nil {
		return fmt.Errorf("declare topology: %w", err)
	}

	// QoS: process one message at a time so multiple replicas
	// (if scaled up later) share work evenly.
	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	deliveries, err := ch.Consume(
		queuePlaced, // queue
		"",          // consumer tag (auto)
		false,       // auto-ack: false — ack only after processing
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	connected.Store(1)
	log.WithField("addr", addr).Info("AMQP connected, consuming order events")

	// Watch the connection's NotifyClose channel so we exit runOnce
	// promptly when RabbitMQ disconnects.
	closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-closeCh:
			if err != nil {
				return fmt.Errorf("amqp connection closed: %w", err)
			}
			return fmt.Errorf("amqp connection closed")
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("deliveries channel closed")
			}
			processDelivery(ch, d)
		}
	}
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		exchangeName, "direct", true, false, false, false, nil,
	); err != nil {
		return err
	}
	for _, q := range []struct {
		name  string
		route string
	}{
		{queuePlaced, routePlaced},
		{queueFulfilled, routeFulfilled},
	} {
		if _, err := ch.QueueDeclare(q.name, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(q.name, q.route, exchangeName, false, nil); err != nil {
			return err
		}
	}
	return nil
}

// processDelivery simulates async fulfillment work, then publishes
// an event to order.fulfilled and acks the original message.
func processDelivery(ch *amqp.Channel, d amqp.Delivery) {
	ordersReceived.Inc()
	start := time.Now()

	var evt orderEvent
	if err := json.Unmarshal(d.Body, &evt); err != nil {
		log.WithError(err).Warn("malformed order event, dropping")
		_ = d.Nack(false, false) // discard
		return
	}

	// Simulate fulfillment processing: random 100–300 ms.
	delay := time.Duration(100+rand.Intn(200)) * time.Millisecond
	time.Sleep(delay)

	if err := ch.Publish(exchangeName, routeFulfilled, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        d.Body,
	}); err != nil {
		log.WithError(err).Warn("publish to order.fulfilled failed; requeueing")
		_ = d.Nack(false, true) // requeue
		return
	}

	if err := d.Ack(false); err != nil {
		log.WithError(err).Warn("ack failed")
		return
	}

	dur := time.Since(start)
	processingDuration.Observe(dur.Seconds())
	ordersProcessed.Inc()
	log.WithFields(log.Fields{
		"order_id":    evt.OrderID,
		"tracking_id": evt.TrackingID,
		"duration_ms": dur.Milliseconds(),
	}).Info("processed order")
}

func serveHTTP() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if connected.Load() == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unhealthy: AMQP not connected"))
	})
	log.WithField("addr", healthPort).Info("HTTP server listening")
	srv := &http.Server{
		Addr:              healthPort,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithError(err).Fatal("HTTP server failed")
	}
}

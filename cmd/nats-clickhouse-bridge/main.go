package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/nats-io/nats.go"
)

type processEvent struct {
	Time       string          `json:"time"`
	ActivityID uint8           `json:"activity_id"`
	Process    json.RawMessage `json:"process"`
	Actor      json.RawMessage `json:"actor"`
	Metadata   json.RawMessage `json:"metadata"`
}

type appConfig struct {
	natsURL   string
	subject   string
	batchSize int
	flushSec  int
	chAddr    string
	chDB      string
	chUser    string
	chPass    string
	table     string
}

func main() {
	configuration := loadConfig()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	natsConn, err := nats.Connect(configuration.natsURL)
	if err != nil {
		fatalf("connect nats: %v", err)
	}
	defer natsConn.Close()

	clickhouseConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{configuration.chAddr},
		Auth: clickhouse.Auth{Database: configuration.chDB, Username: configuration.chUser, Password: configuration.chPass},
	})
	if err != nil {
		fatalf("connect clickhouse: %v", err)
	}

	if err := clickhouseConn.Ping(ctx); err != nil {
		fatalf("ping clickhouse: %v", err)
	}

	events := make(chan processEvent, configuration.batchSize*2)
	subscription, err := natsConn.Subscribe(configuration.subject, func(message *nats.Msg) {
		var event processEvent
		if err := json.Unmarshal(message.Data, &event); err != nil {
			return
		}
		select {
		case events <- event:
		default:
		}
	})
	if err != nil {
		fatalf("subscribe nats: %v", err)
	}
	defer subscription.Unsubscribe()

	ticker := time.NewTicker(time.Duration(configuration.flushSec) * time.Second)
	defer ticker.Stop()

	buffer := make([]processEvent, 0, configuration.batchSize)
	for {
		select {
		case <-ctx.Done():
			if len(buffer) > 0 {
				if err := flushBatch(ctx, clickhouseConn, configuration.table, buffer); err != nil {
					fatalf("flush on shutdown: %v", err)
				}
			}
			fmt.Println("bridge stopped")
			return
		case event := <-events:
			buffer = append(buffer, event)
			if len(buffer) >= configuration.batchSize {
				if err := flushBatch(ctx, clickhouseConn, configuration.table, buffer); err != nil {
					fatalf("flush batch: %v", err)
				}
				buffer = buffer[:0]
			}
		case <-ticker.C:
			if len(buffer) == 0 {
				continue
			}
			if err := flushBatch(ctx, clickhouseConn, configuration.table, buffer); err != nil {
				fatalf("flush ticker: %v", err)
			}
			buffer = buffer[:0]
		}
	}
}

func flushBatch(ctx context.Context, conn clickhouse.Conn, table string, values []processEvent) error {
	query := fmt.Sprintf("INSERT INTO %s (time, activity_id, process, actor, metadata) VALUES", table)
	batch, err := conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}

	for _, value := range values {
		timestamp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value.Time))
		if err != nil {
			timestamp = time.Now().UTC()
		}
		if err := batch.Append(
			timestamp,
			value.ActivityID,
			string(value.Process),
			string(value.Actor),
			string(value.Metadata),
		); err != nil {
			return err
		}
	}

	return batch.Send()
}

func loadConfig() appConfig {
	return appConfig{
		natsURL:   getenv("KUBRIC_NATS_URL", "nats://127.0.0.1:4222"),
		subject:   getenv("KUBRIC_NATS_SUBJECT", "kubric.proc.events"),
		batchSize: getenvInt("KUBRIC_BATCH_SIZE", 10000),
		flushSec:  getenvInt("KUBRIC_FLUSH_SECONDS", 5),
		chAddr:    getenv("KUBRIC_CLICKHOUSE_ADDR", "127.0.0.1:9000"),
		chDB:      getenv("KUBRIC_CLICKHOUSE_DB", "default"),
		chUser:    getenv("KUBRIC_CLICKHOUSE_USER", "default"),
		chPass:    getenv("KUBRIC_CLICKHOUSE_PASS", ""),
		table:     getenv("KUBRIC_CLICKHOUSE_TABLE", "ocsf_process_events"),
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func fatalf(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}

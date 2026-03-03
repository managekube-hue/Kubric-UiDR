package rita

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/ClickHouse/clickhouse-go/v2"
)

type Server struct {
	db     *sql.DB
	router *chi.Mux
	cfg    Config
}

type Config struct {
	MinBeaconConnections uint64
	MinDNSQueries        uint64
	MaxLongConnSecs      uint64
	StrictSchema         bool
}

type Beacon struct {
	SrcIP       string  `json:"src_ip"`
	DstIP       string  `json:"dst_ip"`
	Score       float64 `json:"score"`
	Connections uint64  `json:"connections"`
	AvgBytes    uint64  `json:"avg_bytes"`
	TSScore     float64 `json:"ts_score"`
	DSScore     float64 `json:"ds_score"`
	DurScore    float64 `json:"dur_score"`
}

type DnsTunnel struct {
	FQDN             string  `json:"fqdn"`
	SrcIP            string  `json:"src_ip"`
	Score            float64 `json:"score"`
	QueryCount       uint64  `json:"query_count"`
	UniqueSubdomains uint64  `json:"unique_subdomains"`
}

type LongConnection struct {
	SrcIP         string `json:"src_ip"`
	DstIP         string `json:"dst_ip"`
	DurationSecs  uint64 `json:"duration_secs"`
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
}

func New(clickhouseURL string) (*Server, error) {
	cfg := Config{
		MinBeaconConnections: getenvUint("NDR_RITA_MIN_BEACON_CONNECTIONS", 5),
		MinDNSQueries:        getenvUint("NDR_RITA_MIN_DNS_QUERIES", 20),
		MaxLongConnSecs:      getenvUint("NDR_RITA_MIN_LONG_CONN_SECS", 300),
		StrictSchema:         getenvBool("NDR_RITA_STRICT_SCHEMA", false),
	}

	var db *sql.DB
	var err error
	if clickhouseURL != "" {
		db, err = sql.Open("clickhouse", clickhouseURL)
		if err != nil {
			return nil, fmt.Errorf("rita: open clickhouse: %w", err)
		}
		if err := validateSchema(db, cfg.StrictSchema); err != nil {
			return nil, err
		}
	}

	s := &Server{db: db, cfg: cfg}
	s.router = s.buildRouter()
	return s, nil
}

func (s *Server) Close() {
	if s != nil && s.db != nil {
		_ = s.db.Close()
	}
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/v1/health", s.health)
	r.Get("/api/v1/{tenant}/beacons", s.beacons)
	r.Get("/api/v1/{tenant}/dns/tunneling", s.dnsTunneling)
	r.Get("/api/v1/{tenant}/long-connections", s.longConnections)
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{"status": "ok", "component": "ndr-rita"}
	if s.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.db.PingContext(ctx); err != nil {
			status["status"] = "degraded"
			status["error"] = err.Error()
			writeJSON(w, http.StatusServiceUnavailable, status)
			return
		}
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) beacons(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	rows, err := s.fetchBeaconRows(r.Context(), tenant)
	if err != nil {
		writeJSON(w, http.StatusOK, []Beacon{})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) dnsTunneling(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	rows, err := s.fetchDNSRows(r.Context(), tenant)
	if err != nil {
		writeJSON(w, http.StatusOK, []DnsTunnel{})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) longConnections(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	rows, err := s.fetchLongConnectionRows(r.Context(), tenant)
	if err != nil {
		writeJSON(w, http.StatusOK, []LongConnection{})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) fetchBeaconRows(ctx context.Context, tenant string) ([]Beacon, error) {
	if s.db == nil {
		return []Beacon{}, nil
	}
	const q = `
SELECT
  src_ip,
  dst_ip,
  count() AS connections,
  avg(bytes) AS avg_bytes,
  avg(duration_secs) AS avg_duration,
  stddevPop(duration_secs) AS std_duration
FROM kubric.network_flows
WHERE tenant_id = ?
  AND timestamp >= now() - INTERVAL 1 HOUR
GROUP BY src_ip, dst_ip
HAVING connections >= 5
ORDER BY connections DESC
LIMIT 200`

	rs, err := s.db.QueryContext(ctx, q, tenant)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	out := make([]Beacon, 0)
	for rs.Next() {
		var src, dst string
		var conn uint64
		var avgBytes float64
		var avgDur float64
		var stdDur float64
		if err := rs.Scan(&src, &dst, &conn, &avgBytes, &avgDur, &stdDur); err != nil {
			continue
		}
		if conn < s.cfg.MinBeaconConnections {
			continue
		}

		tsScore := clamp01(1.0 - (stdDur / math.Max(avgDur, 1.0)))
		dsScore := clamp01(math.Log10(math.Max(avgBytes, 1))/6.0)
		durScore := clamp01(avgDur / 300.0)

		score := clamp01((tsScore + dsScore + durScore) / 3.0)

		out = append(out, Beacon{
			SrcIP:       src,
			DstIP:       dst,
			Score:       score,
			Connections: conn,
			AvgBytes:    uint64(math.Max(avgBytes, 0)),
			TSScore:     tsScore,
			DSScore:     dsScore,
			DurScore:    durScore,
		})
	}
	return out, rs.Err()
}

func (s *Server) fetchDNSRows(ctx context.Context, tenant string) ([]DnsTunnel, error) {
	if s.db == nil {
		return []DnsTunnel{}, nil
	}
	const q = `
SELECT
  fqdn,
  src_ip,
  count() AS query_count,
  uniqExact(subdomain) AS unique_subdomains
FROM kubric.network_dns
WHERE tenant_id = ?
  AND timestamp >= now() - INTERVAL 1 HOUR
GROUP BY fqdn, src_ip
HAVING query_count >= 20
ORDER BY unique_subdomains DESC
LIMIT 200`

	rs, err := s.db.QueryContext(ctx, q, tenant)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	out := make([]DnsTunnel, 0)
	for rs.Next() {
		var fqdn, src string
		var queries uint64
		var uniqueSubs uint64
		if err := rs.Scan(&fqdn, &src, &queries, &uniqueSubs); err != nil {
			continue
		}
		if queries < s.cfg.MinDNSQueries {
			continue
		}

		score := clamp01((float64(uniqueSubs)/1000.0 + float64(queries)/5000.0) / 2.0)

		out = append(out, DnsTunnel{
			FQDN:             fqdn,
			SrcIP:            src,
			Score:            score,
			QueryCount:       queries,
			UniqueSubdomains: uniqueSubs,
		})
	}
	return out, rs.Err()
}

func (s *Server) fetchLongConnectionRows(ctx context.Context, tenant string) ([]LongConnection, error) {
	if s.db == nil {
		return []LongConnection{}, nil
	}
	const q = `
SELECT
  src_ip,
  dst_ip,
  max(duration_secs) AS max_duration,
  sum(bytes_out) AS bytes_out,
  sum(bytes_in) AS bytes_in
FROM kubric.network_flows
WHERE tenant_id = ?
  AND timestamp >= now() - INTERVAL 1 HOUR
GROUP BY src_ip, dst_ip
HAVING max_duration >= 300
ORDER BY max_duration DESC
LIMIT 200`

	rs, err := s.db.QueryContext(ctx, q, tenant)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	out := make([]LongConnection, 0)
	for rs.Next() {
		var src, dst string
		var dur uint64
		var outBytes uint64
		var inBytes uint64
		if err := rs.Scan(&src, &dst, &dur, &outBytes, &inBytes); err != nil {
			continue
		}
		if dur < s.cfg.MaxLongConnSecs {
			continue
		}
		out = append(out, LongConnection{
			SrcIP:         src,
			DstIP:         dst,
			DurationSecs:  dur,
			BytesSent:     outBytes,
			BytesReceived: inBytes,
		})
	}
	return out, rs.Err()
}

func validateSchema(db *sql.DB, strict bool) error {
	required := []string{"network_flows", "network_dns"}
	for _, table := range required {
		const q = `SELECT count() FROM system.tables WHERE database = 'kubric' AND name = ?`
		var count uint64
		if err := db.QueryRow(q, table).Scan(&count); err != nil {
			if strict {
				return fmt.Errorf("rita: schema check failed for %s: %w", table, err)
			}
			continue
		}
		if count == 0 && strict {
			return fmt.Errorf("rita: required table missing: kubric.%s", table)
		}
	}
	return nil
}

func getenvUint(key string, fallback uint64) uint64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "TRUE" || v == "True"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

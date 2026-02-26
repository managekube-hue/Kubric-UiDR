//go:build ignore

// Package nocops provides NOC operations tooling.
// K-NOC-INV-001 — Asset Discovery: network scanning and asset inventory management
package nocops

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"
)

type AssetType string

const (
	TypeEndpoint      AssetType = "endpoint"
	TypeServer        AssetType = "server"
	TypeNetworkDevice AssetType = "network_device"
	TypeContainer     AssetType = "container"
	TypeCloud         AssetType = "cloud"
)

// Asset represents a discovered network asset with full inventory details.
type Asset struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	Type           AssetType              `json:"type"`
	Hostname       string                 `json:"hostname"`
	IPAddresses    []string               `json:"ip_addresses"`
	MACAddress     string                 `json:"mac_address,omitempty"`
	OS             string                 `json:"os,omitempty"`
	OSVersion      string                 `json:"os_version,omitempty"`
	Arch           string                 `json:"arch,omitempty"`
	AgentInstalled bool                   `json:"agent_installed"`
	AgentVersion   string                 `json:"agent_version,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	LastSeen       time.Time              `json:"last_seen"`
	FirstSeen      time.Time              `json:"first_seen"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
}

// AssetFilter restricts List queries.
type AssetFilter struct {
	TenantID       string
	Type           AssetType
	AgentInstalled *bool
	Tags           []string
	LastSeenAfter  *time.Time
}

// AssetStore is the persistence abstraction for asset records.
type AssetStore interface {
	Upsert(ctx context.Context, asset Asset) error
	List(ctx context.Context, tenantID string, filter AssetFilter) ([]Asset, error)
	Get(ctx context.Context, id string) (*Asset, error)
	Delete(ctx context.Context, id string) error
}

// AssetDiscovery orchestrates network scanning and inventory synchronisation.
type AssetDiscovery struct {
	store      AssetStore
	osqueryURL string
	nc         *nats.Conn
}

// NewAssetDiscovery constructs an AssetDiscovery with a backing store, an optional
// osquery gRPC endpoint URL, and an optional NATS connection for event publishing.
func NewAssetDiscovery(store AssetStore, osqueryURL string, nc *nats.Conn) *AssetDiscovery {
	return &AssetDiscovery{store: store, osqueryURL: osqueryURL, nc: nc}
}

// DiscoverViaARP sends reverse-DNS probes to discover live hosts in a CIDR.
// It caps scanning to 1 024 addresses to prevent unintentional runaway scans.
func (ad *AssetDiscovery) DiscoverViaARP(ctx context.Context, cidr string) ([]Asset, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	var assets []Asset
	ips := hostsInCIDR(ipNet)

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return assets, ctx.Err()
		default:
		}

		host := ip.String()
		names, _ := net.LookupAddr(host)
		hostname := host
		if len(names) > 0 {
			hostname = strings.TrimSuffix(names[0], ".")
		}

		asset := Asset{
			ID:          fmt.Sprintf("arp-%s", strings.ReplaceAll(host, ".", "-")),
			Type:        TypeEndpoint,
			Hostname:    hostname,
			IPAddresses: []string{host},
			FirstSeen:   time.Now().UTC(),
			LastSeen:    time.Now().UTC(),
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

// SyncInventory discovers assets for a tenant, upserts them into the store,
// and returns counts of newly added and updated records.
func (ad *AssetDiscovery) SyncInventory(ctx context.Context, tenantID string) (added, updated int, err error) {
	existing, err := ad.store.List(ctx, tenantID, AssetFilter{TenantID: tenantID})
	if err != nil {
		return 0, 0, fmt.Errorf("list existing assets: %w", err)
	}
	existingMap := make(map[string]struct{}, len(existing))
	for _, a := range existing {
		existingMap[a.ID] = struct{}{}
	}

	// In a real deployment the CIDR list would be retrieved from tenant config.
	// Here we illustrate the upsert logic with a synthetic example.
	discovered, err := ad.DiscoverViaARP(ctx, "192.168.0.0/24")
	if err != nil {
		return added, updated, fmt.Errorf("arp discovery: %w", err)
	}

	for i := range discovered {
		discovered[i].TenantID = tenantID
		if upsertErr := ad.store.Upsert(ctx, discovered[i]); upsertErr != nil {
			return added, updated, fmt.Errorf("upsert asset %s: %w", discovered[i].ID, upsertErr)
		}
		if _, exists := existingMap[discovered[i].ID]; exists {
			updated++
		} else {
			added++
			_ = ad.PublishToNATS(tenantID, discovered[i])
		}
	}
	return added, updated, nil
}

// ExportToCSV writes assets as CSV rows to the supplied writer.
func (ad *AssetDiscovery) ExportToCSV(w io.Writer, assets []Asset) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{
		"id", "hostname", "type", "ip_addresses", "os", "agent_installed", "last_seen",
	}); err != nil {
		return err
	}
	for _, a := range assets {
		if err := cw.Write([]string{
			a.ID,
			a.Hostname,
			string(a.Type),
			strings.Join(a.IPAddresses, ";"),
			a.OS,
			fmt.Sprintf("%v", a.AgentInstalled),
			a.LastSeen.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	return cw.Error()
}

// Statistics returns a type→count breakdown plus a "total" key.
func (ad *AssetDiscovery) Statistics(ctx context.Context, tenantID string) (map[string]int, error) {
	assets, err := ad.store.List(ctx, tenantID, AssetFilter{TenantID: tenantID})
	if err != nil {
		return nil, err
	}
	stats := make(map[string]int)
	for _, a := range assets {
		stats[string(a.Type)]++
	}
	stats["total"] = len(assets)
	return stats, nil
}

// PublishToNATS publishes an asset discovery event on the tenant-scoped subject
// kubric.<tenantID>.inventory.asset.v1.
func (ad *AssetDiscovery) PublishToNATS(tenantID string, asset Asset) error {
	if ad.nc == nil {
		return nil
	}
	data, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("marshal asset: %w", err)
	}
	subject := fmt.Sprintf("kubric.%s.inventory.asset.v1", tenantID)
	if err := ad.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("nats publish %s: %w", subject, err)
	}
	return nil
}

// ---- low-level IP helpers ----

func hostsInCIDR(network *net.IPNet) []net.IP {
	var ips []net.IP
	for ip := cloneIP(network.IP.Mask(network.Mask)); network.Contains(ip); incrementIP(ip) {
		last := ip[len(ip)-1]
		if last == 0 || last == 255 { // skip network address and broadcast
			continue
		}
		ips = append(ips, cloneIP(ip))
		if len(ips) >= 1024 { // cap to avoid runaway scans
			break
		}
	}
	return ips
}

func cloneIP(ip net.IP) net.IP {
	c := make(net.IP, len(ip))
	copy(c, ip)
	return c
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// ---- sort helpers ----

// AssetSlice implements sort.Interface over a slice of Asset, ordering by Hostname.
type AssetSlice []Asset

func (s AssetSlice) Len() int           { return len(s) }
func (s AssetSlice) Less(i, j int) bool { return s[i].Hostname < s[j].Hostname }
func (s AssetSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// SortAssets sorts assets alphabetically by Hostname in place.
func SortAssets(assets []Asset) { sort.Sort(AssetSlice(assets)) }

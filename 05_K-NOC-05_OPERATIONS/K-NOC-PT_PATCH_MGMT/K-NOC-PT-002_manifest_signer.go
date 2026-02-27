// Package noc provides NOC operations tooling.
// K-NOC-PT-002 — Patch Manifest Signer: sign patch manifests with Ed25519 for tamper-proof patch authorization.
package noc

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// PatchEntry describes a single package to be patched.
type PatchEntry struct {
	PackageName   string `json:"package_name"`
	TargetVersion string `json:"target_version"`
	Checksum      string `json:"checksum"`
	DownloadURL   string `json:"download_url"`
}

// PatchManifest is the unsigned manifest payload.
type PatchManifest struct {
	ManifestID  string       `json:"manifest_id"`
	AssetID     string       `json:"asset_id"`
	TenantID    string       `json:"tenant_id"`
	Patches     []PatchEntry `json:"patches"`
	AuthorizedBy string      `json:"authorized_by"`
	ExpiresAt   time.Time    `json:"expires_at"`
	CreatedAt   time.Time    `json:"created_at"`
}

// SignedManifest wraps a PatchManifest with an Ed25519 signature.
type SignedManifest struct {
	Manifest     PatchManifest `json:"manifest"`
	Signature    string        `json:"signature"`     // base64-encoded Ed25519 signature
	PublicKeyHex string        `json:"public_key_hex"` // hex-encoded public key for verification
}

// ManifestSigner holds an Ed25519 key pair and signs/verifies patch manifests.
type ManifestSigner struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// GenerateKeyPair generates a new Ed25519 key pair.
func GenerateKeyPair() (*ManifestSigner, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key pair: %w", err)
	}
	return &ManifestSigner{privateKey: priv, publicKey: pub}, nil
}

// NewManifestSigner loads an Ed25519 private key from the given PEM file.
func NewManifestSigner(privateKeyPath string) (*ManifestSigner, error) {
	if privateKeyPath == "" {
		privateKeyPath = os.Getenv("PATCH_SIGNING_KEY_PATH")
	}
	priv, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}
	return &ManifestSigner{
		privateKey: priv,
		publicKey:  priv.Public().(ed25519.PublicKey),
	}, nil
}

// Sign serialises the manifest to JSON and signs it with the Ed25519 private key.
func (ms *ManifestSigner) Sign(manifest PatchManifest) (SignedManifest, error) {
	data, err := json.Marshal(manifest)
	if err != nil {
		return SignedManifest{}, fmt.Errorf("marshal manifest: %w", err)
	}

	sig := ed25519.Sign(ms.privateKey, data)

	return SignedManifest{
		Manifest:     manifest,
		Signature:    base64.StdEncoding.EncodeToString(sig),
		PublicKeyHex: hex.EncodeToString(ms.publicKey),
	}, nil
}

// Verify checks the signature on a SignedManifest against the provided public key.
func (ms *ManifestSigner) Verify(signed SignedManifest, pubKey ed25519.PublicKey) (bool, error) {
	data, err := json.Marshal(signed.Manifest)
	if err != nil {
		return false, fmt.Errorf("marshal manifest for verification: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signed.Signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}

	return ed25519.Verify(pubKey, data, sigBytes), nil
}

// SaveKeyPair writes the key pair to PEM-encoded files.
func (ms *ManifestSigner) SaveKeyPair(privateKeyPath, publicKeyPath string) error {
	// Marshal private key.
	privDER, err := x509.MarshalPKCS8PrivateKey(ms.privateKey)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	if err := os.WriteFile(privateKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("write private key to %s: %w", privateKeyPath, err)
	}

	// Marshal public key.
	pubDER, err := x509.MarshalPKIXPublicKey(ms.publicKey)
	if err != nil {
		return fmt.Errorf("marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	if err := os.WriteFile(publicKeyPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("write public key to %s: %w", publicKeyPath, err)
	}
	return nil
}

// LoadPrivateKey loads an Ed25519 private key from a PEM file (exported for external callers).
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	return loadPrivateKey(path)
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key file %s: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key from %s: %w", path, err)
	}

	ed25519Key, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key in %s is not Ed25519 (got %T)", path, key)
	}
	return ed25519Key, nil
}

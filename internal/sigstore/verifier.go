// Package sigstore provides container image signature verification using
// the Sigstore ecosystem (Cosign/Rekor).  It verifies that container images
// deployed across Kubric's kind cluster are signed with a trusted identity,
// supporting the Sigstore-Cosign compliance framework in the GRC registry.
//
// Depends on: github.com/sigstore/sigstore
package sigstore

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
)

// VerifyResult holds the outcome of an image signature verification.
type VerifyResult struct {
	ImageRef   string    `json:"image_ref"`
	Verified   bool      `json:"verified"`
	SignerID   string    `json:"signer_id,omitempty"`
	VerifiedAt time.Time `json:"verified_at"`
	Error      string    `json:"error,omitempty"`
}

// Verifier checks container image signatures against a trusted public key.
type Verifier struct {
	pubKey   crypto.PublicKey
	verifier signature.Verifier
	keyPath  string
}

// NewVerifier loads a PEM-encoded ECDSA public key for signature verification.
// keyPath points to the cosign public key, e.g. "/etc/kubric/cosign.pub".
// Falls back to COSIGN_PUB_KEY env var.
func NewVerifier(keyPath string) (*Verifier, error) {
	if keyPath == "" {
		keyPath = os.Getenv("COSIGN_PUB_KEY")
	}
	if keyPath == "" {
		return nil, nil // disabled — caller checks nil
	}
	pemData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("sigstore read key %s: %w", keyPath, err)
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("sigstore: no PEM block in %s", keyPath)
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("sigstore parse key: %w", err)
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("sigstore: expected ECDSA public key, got %T", pub)
	}
	verifier, err := signature.LoadECDSAVerifier(ecPub, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("sigstore load verifier: %w", err)
	}
	return &Verifier{
		pubKey:   ecPub,
		verifier: verifier,
		keyPath:  keyPath,
	}, nil
}

// VerifyBlob checks that data matches the provided signature bytes against
// the loaded public key.  Used for verifying SBOM or attestation signatures.
func (v *Verifier) VerifyBlob(ctx context.Context, data, sig []byte) error {
	if v == nil {
		return fmt.Errorf("sigstore verifier not initialised")
	}
	return v.verifier.VerifySignature(
		bytes_reader(sig),
		bytes_reader(data),
	)
}

// VerifyImageSignature performs a high-level image signature verification.
// imageRef is the full image reference, e.g. "ghcr.io/managekube-hue/kubric-kic:latest".
// sigPayload is the raw cosign signature payload (from Rekor or an OCI signature).
// sigBytes is the detached signature.
func (v *Verifier) VerifyImageSignature(_ context.Context, imageRef string, sigPayload, sigBytes []byte) *VerifyResult {
	result := &VerifyResult{
		ImageRef:   imageRef,
		VerifiedAt: time.Now().UTC(),
	}
	if v == nil {
		result.Error = "verifier disabled (no public key configured)"
		return result
	}
	err := v.verifier.VerifySignature(
		bytes_reader(sigBytes),
		bytes_reader(sigPayload),
	)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Verified = true
	result.SignerID = v.keyPath
	return result
}

// Close is a no-op — satisfies the integration pattern.
func (v *Verifier) Close() {}

// bytes_reader wraps a byte slice into a signature.MessageReader.
func bytes_reader(b []byte) signature.MessageReader {
	return &bytesReader{data: b, pos: 0}
}

// bytesReader implements io.Reader for signature verification.
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type sourceSpec struct {
	Name   string
	URL    string
	OutDir string
	SHA256 string
}

func main() {
	var verify bool
	flag.BoolVar(&verify, "sha-check", true, "verify SHA256 for pinned artifacts where configured")
	flag.Parse()

	repoRoot, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	sources := []sourceSpec{
		{
			Name:   "nDPI 5.0",
			URL:    "https://github.com/ntop/nDPI/archive/refs/tags/5.0.tar.gz",
			OutDir: filepath.Join(repoRoot, "vendor", "sources"),
			SHA256: "",
		},
		{
			Name:   "Suricata ET Open",
			URL:    "https://rules.emergingthreats.net/open/suricata-7.0/emerging.rules.tar.gz",
			OutDir: filepath.Join(repoRoot, "vendor", "sources"),
			SHA256: "",
		},
	}

	for _, src := range sources {
		if err := os.MkdirAll(src.OutDir, 0o755); err != nil {
			fatal(err)
		}
		artifactPath, err := download(src)
		if err != nil {
			fatal(fmt.Errorf("%s download failed: %w", src.Name, err))
		}
		if verify && src.SHA256 != "" {
			if err := verifySHA256(artifactPath, src.SHA256); err != nil {
				fatal(fmt.Errorf("%s checksum failed: %w", src.Name, err))
			}
		}
		fmt.Printf("downloaded: %s -> %s\n", src.Name, artifactPath)

		if strings.Contains(filepath.Base(artifactPath), ".tar.gz") {
			dst := filepath.Join(repoRoot, "vendor", "_staging", strings.TrimSuffix(filepath.Base(artifactPath), ".tar.gz"))
			if err := extractTarGz(artifactPath, dst); err != nil {
				fatal(fmt.Errorf("%s extract failed: %w", src.Name, err))
			}
			fmt.Printf("extracted: %s -> %s\n", src.Name, dst)
		}
	}

	if err := ensureVendorPaths(repoRoot); err != nil {
		fatal(err)
	}

	fmt.Println("vendor update completed")
}

func ensureVendorPaths(repoRoot string) error {
	paths := []string{
		filepath.Join(repoRoot, "vendor", "ndpi"),
		filepath.Join(repoRoot, "vendor", "zeek", "policy", "protocols", "ssl"),
		filepath.Join(repoRoot, "vendor", "coreruleset", "rules"),
		filepath.Join(repoRoot, "vendor", "suricata"),
	}
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func download(src sourceSpec) (string, error) {
	resp, err := http.Get(src.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}

	fileName := filepath.Base(src.URL)
	if fileName == "." || fileName == "/" || fileName == "" {
		fileName = strings.ReplaceAll(src.Name, " ", "-") + ".tar.gz"
	}
	outPath := filepath.Join(src.OutDir, fileName)
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return outPath, nil
}

func verifySHA256(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(sum, expectedHex) {
		return fmt.Errorf("expected %s got %s", expectedHex, sum)
	}
	return nil
}

func extractTarGz(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dst, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

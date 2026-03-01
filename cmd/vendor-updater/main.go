package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	interval := getEnv("UPDATE_INTERVAL", "24h")
	duration, _ := time.ParseDuration(interval)
	vendorDir := getEnv("VENDOR_DIR", "/vendor")

	log.Printf("Vendor updater started (interval: %s)", interval)

	for {
		log.Println("Updating vendor rules...")
		
		if err := updateSigmaRules(vendorDir); err != nil {
			log.Printf("Sigma update failed: %v", err)
		}
		
		if err := updateYaraRules(vendorDir); err != nil {
			log.Printf("YARA update failed: %v", err)
		}
		
		if err := updateSuricataRules(vendorDir); err != nil {
			log.Printf("Suricata update failed: %v", err)
		}
		
		log.Printf("Update complete. Next in %s", interval)
		time.Sleep(duration)
	}
}

func updateSigmaRules(vendorDir string) error {
	sigmaDir := filepath.Join(vendorDir, "sigma")
	repo := getEnv("SIGMA_REPO", "https://github.com/SigmaHQ/sigma")
	
	if _, err := os.Stat(sigmaDir); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--depth", "1", repo, sigmaDir)
		return cmd.Run()
	}
	
	cmd := exec.Command("git", "-C", sigmaDir, "pull", "--depth", "1")
	return cmd.Run()
}

func updateYaraRules(vendorDir string) error {
	yaraDir := filepath.Join(vendorDir, "yara-rules")
	repo := getEnv("YARA_REPO", "https://github.com/Yara-Rules/rules")
	
	if _, err := os.Stat(yaraDir); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--depth", "1", repo, yaraDir)
		return cmd.Run()
	}
	
	cmd := exec.Command("git", "-C", yaraDir, "pull", "--depth", "1")
	return cmd.Run()
}

func updateSuricataRules(vendorDir string) error {
	suricataDir := filepath.Join(vendorDir, "suricata-rules")
	url := getEnv("SURICATA_URL", "https://rules.emergingthreats.net/open/suricata-6.0/emerging.rules.tar.gz")
	
	os.MkdirAll(suricataDir, 0755)
	
	cmd := exec.Command("curl", "-L", "-o", filepath.Join(suricataDir, "rules.tar.gz"), url)
	if err := cmd.Run(); err != nil {
		return err
	}
	
	cmd = exec.Command("tar", "-xzf", filepath.Join(suricataDir, "rules.tar.gz"), "-C", suricataDir)
	return cmd.Run()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

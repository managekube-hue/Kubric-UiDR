// handler_supply_chain.go — KIC HTTP handlers for OpenSSF Scorecard and
// Sigstore image signature verification.
package kic

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/managekube-hue/Kubric-UiDR/internal/scorecard"
	kubricsig "github.com/managekube-hue/Kubric-UiDR/internal/sigstore"
)

type supplyChainHandler struct {
	sc  *scorecard.Runner
	sig *kubricsig.Verifier
}

func newSupplyChainHandler(sc *scorecard.Runner, sig *kubricsig.Verifier) *supplyChainHandler {
	return &supplyChainHandler{sc: sc, sig: sig}
}

// runScorecard runs OpenSSF Scorecard against one or more repositories.
//
//	POST /supply-chain/scorecard
//	Body: {"repos": ["github.com/owner/repo"]}
func (h *supplyChainHandler) runScorecard(w http.ResponseWriter, r *http.Request) {
	if h.sc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "scorecard runner not configured (set GITHUB_AUTH_TOKEN)",
		})
		return
	}
	var body struct {
		Repos []string `json:"repos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(body.Repos) == 0 {
		writeError(w, http.StatusBadRequest, "repos array is required")
		return
	}
	if len(body.Repos) > 10 {
		writeError(w, http.StatusBadRequest, "max 10 repos per request")
		return
	}
	results, err := h.sc.ScoreMultiple(r.Context(), body.Repos)
	if err != nil {
		// Partial results are still returned alongside the error
		writeJSON(w, http.StatusOK, map[string]any{
			"results": results,
			"warning": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

// verifyImage verifies a container image signature using Sigstore/Cosign.
//
//	POST /supply-chain/verify-image
//	Body: {"image_ref": "ghcr.io/...:tag", "payload": "<base64>", "signature": "<base64>"}
func (h *supplyChainHandler) verifyImage(w http.ResponseWriter, r *http.Request) {
	if h.sig == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "sigstore verifier not configured (set COSIGN_PUB_KEY)",
		})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}
	var req struct {
		ImageRef  string `json:"image_ref"`
		Payload   []byte `json:"payload"`
		Signature []byte `json:"signature"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.ImageRef == "" {
		writeError(w, http.StatusBadRequest, "image_ref is required")
		return
	}
	result := h.sig.VerifyImageSignature(r.Context(), req.ImageRef, req.Payload, req.Signature)
	status := http.StatusOK
	if !result.Verified {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, result)
}

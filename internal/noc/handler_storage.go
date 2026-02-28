// handler_storage.go — NOC HTTP handlers for MinIO object store operations.
package noc

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/managekube-hue/Kubric-UiDR/internal/storage"
)

type storageHandler struct {
	store *storage.ObjectStore
}

func (h *storageHandler) upload(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	key := chi.URLParam(r, "key")
	if bucket == "" || key == "" {
		writeError(w, http.StatusBadRequest, "bucket and key required")
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 50<<20)) // 50MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	meta, err := h.store.PutObject(r.Context(), bucket, key, data, ct)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, meta)
}

func (h *storageHandler) download(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	key := chi.URLParam(r, "key")
	data, meta, err := h.store.GetObject(r.Context(), bucket, key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("ETag", meta.ETag)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *storageHandler) list(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	prefix := r.URL.Query().Get("prefix")
	objects, err := h.store.ListObjects(r.Context(), bucket, prefix)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, objects)
}

func (h *storageHandler) presign(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	key := chi.URLParam(r, "key")
	url, err := h.store.PresignedGet(r.Context(), bucket, key, 15*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url, "expires_in": "15m"})
}

func (h *storageHandler) deleteObj(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	key := chi.URLParam(r, "key")
	if err := h.store.DeleteObject(r.Context(), bucket, key); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// bucketStats returns summary info for all default buckets
func (h *storageHandler) bucketStats(w http.ResponseWriter, r *http.Request) {
	type bucketInfo struct {
		Bucket string `json:"bucket"`
		Count  int    `json:"object_count"`
	}
	buckets := []string{storage.BucketEvidence, storage.BucketSBOM, storage.BucketScans, storage.BucketBackups}
	var info []bucketInfo
	for _, b := range buckets {
		objs, err := h.store.ListObjects(r.Context(), b, "")
		if err != nil {
			info = append(info, bucketInfo{Bucket: b, Count: -1})
			continue
		}
		info = append(info, bucketInfo{Bucket: b, Count: len(objs)})
	}
	_ = json.NewEncoder(w)
	writeJSON(w, http.StatusOK, info)
}

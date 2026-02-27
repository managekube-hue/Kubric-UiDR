"""
K-KAI-ML-007_hikari_preprocessor.py
asyncpg data preprocessor that fetches raw OCSF events from PostgreSQL
and preprocesses them into ML feature vectors.
"""

import asyncio
import hashlib
import logging
import math
import os
from datetime import datetime
from typing import Optional, Tuple

import numpy as np

logger = logging.getLogger(__name__)

try:
    import asyncpg
    _ASYNCPG_AVAILABLE = True
except ImportError:
    _ASYNCPG_AVAILABLE = False

try:
    from sklearn.feature_extraction.text import TfidfVectorizer
    from sklearn.preprocessing import LabelEncoder, OneHotEncoder
    _SKLEARN_AVAILABLE = True
except ImportError:
    _SKLEARN_AVAILABLE = False

DATABASE_URL = os.environ.get(
    "DATABASE_URL", "postgresql://kai:kai@localhost:5432/kubric"
)

# Feature dimensions
IP_HASH_DIM = 32
TFIDF_FEATURES = 128
_PROTOCOLS = ["tcp", "udp", "icmp", "dns", "http", "https", "ftp", "ssh", "smb", "rdp"]


def _ip_to_vec(ip: str, dim: int = IP_HASH_DIM) -> np.ndarray:
    """Hash an IP address string into a float vector of length dim."""
    raw = hashlib.sha256(ip.encode()).digest()
    # Map bytes into [-1, 1]
    arr = np.frombuffer(raw, dtype=np.uint8).astype(np.float32)
    arr = arr / 127.5 - 1.0
    # Tile or truncate to requested dimension
    if dim <= len(arr):
        return arr[:dim]
    reps = math.ceil(dim / len(arr))
    return np.tile(arr, reps)[:dim]


def _log_scale(value: Optional[int]) -> float:
    """Normalise a PID/PPID-like integer to log scale in [0, 1]."""
    if not value or value <= 0:
        return 0.0
    return math.log1p(value) / math.log1p(65535)


def _one_hot_protocol(proto: Optional[str]) -> np.ndarray:
    """One-hot encode a protocol string into a fixed-length vector."""
    proto = (proto or "").lower().strip()
    vec = np.zeros(len(_PROTOCOLS), dtype=np.float32)
    if proto in _PROTOCOLS:
        vec[_PROTOCOLS.index(proto)] = 1.0
    return vec


class HikariPreprocessor:
    """
    Fetches OCSF events from Postgres and converts them to numpy feature matrices
    ready for scikit-learn / PyTorch training pipelines.

    Usage::

        prep = HikariPreprocessor()
        X, y = asyncio.run(prep.preprocess_process_events(start, end))
        X_net, y_net = asyncio.run(prep.preprocess_network_events(start, end))
    """

    def __init__(self, dsn: str = DATABASE_URL) -> None:
        self._dsn = dsn
        self._pool: Optional[object] = None
        self._tfidf: Optional[object] = None

    # ------------------------------------------------------------------
    # Connection management
    # ------------------------------------------------------------------

    async def _get_pool(self):
        if self._pool is None:
            if not _ASYNCPG_AVAILABLE:
                raise RuntimeError("asyncpg not installed — pip install asyncpg")
            self._pool = await asyncpg.create_pool(
                dsn=self._dsn, min_size=2, max_size=10
            )
        return self._pool

    # ------------------------------------------------------------------
    # Process events (class_uid=4001 — Process Activity)
    # ------------------------------------------------------------------

    async def preprocess_process_events(
        self, start: datetime, end: datetime
    ) -> Tuple[np.ndarray, np.ndarray]:
        """
        Query process activity events and build feature matrix.

        Features per row:
          - TF-IDF on cmdline (128 dims)
          - log_scale(pid), log_scale(ppid)  (2 dims)
          - ip hash of src_ip (32 dims)
          Total: 162 dims

        Labels: 1 if severity_id >= 4 else 0
        """
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT cmdline, pid, ppid, src_ip, severity_id
                FROM ocsf_events
                WHERE class_uid = 4001
                  AND time BETWEEN $1 AND $2
                ORDER BY time
                """,
                start,
                end,
            )

        if not rows:
            return np.empty((0, 162), dtype=np.float32), np.empty((0,), dtype=np.int32)

        cmdlines = [r["cmdline"] or "" for r in rows]
        pids = [r["pid"] for r in rows]
        ppids = [r["ppid"] for r in rows]
        src_ips = [r["src_ip"] or "0.0.0.0" for r in rows]
        labels = np.array([1 if (r["severity_id"] or 0) >= 4 else 0 for r in rows], dtype=np.int32)

        # TF-IDF
        tfidf = TfidfVectorizer(max_features=TFIDF_FEATURES, analyzer="char_wb", ngram_range=(3, 5))
        tfidf_mat = tfidf.fit_transform(cmdlines).toarray().astype(np.float32)
        self._tfidf = tfidf

        # Numeric feats
        pid_feats = np.array([[_log_scale(p), _log_scale(pp)] for p, pp in zip(pids, ppids)], dtype=np.float32)

        # IP hash
        ip_feats = np.array([_ip_to_vec(ip) for ip in src_ips], dtype=np.float32)

        X = np.concatenate([tfidf_mat, pid_feats, ip_feats], axis=1)
        logger.info("preprocess_process_events — rows=%d X.shape=%s", len(rows), X.shape)
        return X, labels

    # ------------------------------------------------------------------
    # Network events (class_uid=4003 — Network Activity)
    # ------------------------------------------------------------------

    async def preprocess_network_events(
        self, start: datetime, end: datetime
    ) -> Tuple[np.ndarray, np.ndarray]:
        """
        Query network activity events and build feature matrix.

        Features per row:
          - one-hot protocol  (10 dims)
          - ip hash(src_ip)   (32 dims)
          - ip hash(dst_ip)   (32 dims)
          - log(bytes_in/65535), log(bytes_out/65535), log(port/65535)  (3 dims)
          Total: 77 dims

        Labels: 1 if severity_id >= 4 else 0
        """
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT src_ip, dst_ip, protocol, dst_port,
                       bytes_in, bytes_out, severity_id
                FROM ocsf_events
                WHERE class_uid = 4003
                  AND time BETWEEN $1 AND $2
                ORDER BY time
                """,
                start,
                end,
            )

        if not rows:
            return np.empty((0, 77), dtype=np.float32), np.empty((0,), dtype=np.int32)

        labels = np.array([1 if (r["severity_id"] or 0) >= 4 else 0 for r in rows], dtype=np.int32)

        feats = []
        for r in rows:
            proto_vec = _one_hot_protocol(r["protocol"])
            src_vec = _ip_to_vec(r["src_ip"] or "0.0.0.0")
            dst_vec = _ip_to_vec(r["dst_ip"] or "0.0.0.0")
            numerics = np.array([
                _log_scale(r["bytes_in"] or 0),
                _log_scale(r["bytes_out"] or 0),
                _log_scale(r["dst_port"] or 0),
            ], dtype=np.float32)
            row_vec = np.concatenate([proto_vec, src_vec, dst_vec, numerics])
            feats.append(row_vec)

        X = np.array(feats, dtype=np.float32)
        logger.info("preprocess_network_events — rows=%d X.shape=%s", len(rows), X.shape)
        return X, labels

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()
            self._pool = None

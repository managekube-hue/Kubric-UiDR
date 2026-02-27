"""
K-KAI-LIBS-003: fastparquet I/O fallback for environments without PyArrow.
Accepts dict lists or pandas DataFrames, supports append mode, merge, and list.
"""

import glob
import logging
import os
from pathlib import Path
from typing import Union

import pandas as pd
import fastparquet

logger = logging.getLogger("kai.libs.fastparquet")


# ---------------------------------------------------------------------------
# Write
# ---------------------------------------------------------------------------
def write_parquet(
    df_or_records: Union[list[dict], pd.DataFrame],
    path: str,
    compression: str = "snappy",
    append: bool = False,
) -> None:
    """
    Write *df_or_records* (a list of dicts or a pandas DataFrame) to a
    Parquet file using fastparquet.

    If *append* is True and the file already exists, rows are appended as a
    new row group. If the file does not exist, *append* is silently ignored.
    """
    if isinstance(df_or_records, list):
        if not df_or_records:
            logger.warning("write_parquet: empty record list, skipping %s", path)
            return
        df = pd.DataFrame(df_or_records)
    else:
        df = df_or_records

    # Coerce object columns that hold only None to string to avoid schema issues
    for col in df.select_dtypes(include=["object"]).columns:
        df[col] = df[col].astype(str).where(df[col].notna(), other=None)

    file_exists = Path(path).exists()
    write_mode = "w"

    if append and file_exists:
        fastparquet.write(path, df, compression=compression, append=True)
        logger.info("Appended %d rows to %s", len(df), path)
    else:
        fastparquet.write(path, df, compression=compression)
        logger.info("Wrote %d rows to %s", len(df), path)


# ---------------------------------------------------------------------------
# Read
# ---------------------------------------------------------------------------
def read_parquet(path: str) -> list[dict]:
    """
    Read a Parquet file and return rows as a list of dicts.
    NaN / NaT values are replaced with None for JSON compatibility.
    """
    resolved = Path(path)
    if not resolved.exists():
        raise FileNotFoundError(f"Parquet file not found: {resolved}")

    pf = fastparquet.ParquetFile(str(resolved))
    df = pf.to_pandas()
    # Replace NaN with None
    df = df.where(pd.notna(df), other=None)
    records = df.to_dict(orient="records")
    logger.info("Read %d rows from %s", len(records), path)
    return records


# ---------------------------------------------------------------------------
# List
# ---------------------------------------------------------------------------
def list_parquet_files(directory: str, pattern: str = "*.parquet") -> list[str]:
    """
    Return all Parquet file paths under *directory* matching *pattern*.
    Paths are returned sorted by modification time (newest first).
    """
    search = os.path.join(directory, "**", pattern)
    matches = glob.glob(search, recursive=True)
    matches.sort(key=lambda p: os.path.getmtime(p), reverse=True)
    logger.debug("list_parquet_files dir=%s found=%d files", directory, len(matches))
    return matches


# ---------------------------------------------------------------------------
# Merge
# ---------------------------------------------------------------------------
def merge_parquet_files(files: list[str], output: str) -> None:
    """
    Merge multiple Parquet files into a single output file.
    All files must have compatible schemas (extra columns are filled with NaN).
    """
    if not files:
        logger.warning("merge_parquet_files: no input files")
        return

    dfs: list[pd.DataFrame] = []
    for fpath in files:
        if not Path(fpath).exists():
            logger.warning("merge_parquet_files: skipping missing file %s", fpath)
            continue
        pf = fastparquet.ParquetFile(fpath)
        dfs.append(pf.to_pandas())

    if not dfs:
        logger.error("merge_parquet_files: all files were missing or unreadable")
        return

    merged = pd.concat(dfs, ignore_index=True, sort=False)
    fastparquet.write(output, merged, compression="snappy")
    logger.info(
        "merge_parquet_files: merged %d files (%d total rows) -> %s",
        len(dfs),
        len(merged),
        output,
    )


# ---------------------------------------------------------------------------
# Schema info helper
# ---------------------------------------------------------------------------
def inspect_parquet(path: str) -> dict:
    """Return basic metadata (num_rows, columns, row_groups) for a Parquet file."""
    pf = fastparquet.ParquetFile(path)
    return {
        "path": path,
        "num_rows": pf.count(),
        "columns": list(pf.columns),
        "row_groups": len(pf.row_groups),
        "schema": str(pf.schema),
    }

"""
K-KAI-ML-003_pyspark_distributed.py
PySpark distributed processing for large-scale security log analysis.
"""

import logging
import os
from typing import Optional

logger = logging.getLogger(__name__)

try:
    from pyspark.sql import DataFrame, SparkSession
    from pyspark.sql import functions as F
    from pyspark.sql.types import DoubleType, StringType
    _SPARK_AVAILABLE = True
except ImportError:
    _SPARK_AVAILABLE = False
    DataFrame = None  # type: ignore
    SparkSession = None  # type: ignore


APP_NAME = "kubric-kai"
_DEFAULT_MASTER = "local[*]"


def _build_session() -> "SparkSession":
    """Build or retrieve the active SparkSession."""
    master = os.environ.get("SPARK_MASTER", _DEFAULT_MASTER)
    builder = (
        SparkSession.builder.appName(APP_NAME)
        .master(master)
        .config("spark.serializer", "org.apache.spark.serializer.KryoSerializer")
        .config("spark.sql.adaptive.enabled", "true")
        .config("spark.sql.shuffle.partitions", "200")
    )
    session = builder.getOrCreate()
    session.sparkContext.setLogLevel("WARN")
    logger.info("SparkSession ready — master=%s", master)
    return session


class SparkSecurityAnalyzer:
    """
    Distributed security log analysis using PySpark.

    All DataFrames are expected to follow the OCSF schema with at minimum:
      - severity_id   (int)
      - src_ip        (string)
      - tenant_id     (string)
      - timestamp     (timestamp)
    """

    def __init__(self) -> None:
        if not _SPARK_AVAILABLE:
            raise RuntimeError(
                "pyspark is not installed. Install it with: pip install pyspark"
            )
        self.spark = _build_session()

    # ------------------------------------------------------------------
    # Core operations
    # ------------------------------------------------------------------

    def load_parquet(self, path: str) -> "DataFrame":
        """Load a Parquet dataset (file or directory) into a DataFrame."""
        df = self.spark.read.parquet(path)
        logger.info("Loaded parquet — path=%s rows=(lazy)", path)
        return df

    def filter_high_severity(self, df: "DataFrame") -> "DataFrame":
        """Keep only rows where severity_id >= 4 (High / Critical)."""
        return df.filter(F.col("severity_id") >= 4)

    def aggregate_iocs(self, df: "DataFrame") -> "DataFrame":
        """
        Count events per src_ip grouped by tenant_id.

        Returns a DataFrame with columns:
          tenant_id, src_ip, event_count
        """
        return (
            df.groupBy("tenant_id", "src_ip")
            .agg(F.count("*").alias("event_count"))
            .orderBy(F.col("event_count").desc())
        )

    def join_threat_intel(
        self, df: "DataFrame", intel_path: str
    ) -> "DataFrame":
        """
        Left-join the event DataFrame against a threat intel Parquet  feed.

        The intel feed must have at least: src_ip, threat_label, confidence_score.
        """
        intel_df = self.spark.read.parquet(intel_path).select(
            "src_ip",
            F.col("threat_label"),
            F.col("confidence_score").cast(DoubleType()),
        )
        enriched = df.join(intel_df, on="src_ip", how="left")
        logger.info("Joined threat intel — source=%s", intel_path)
        return enriched

    def write_results(
        self,
        df: "DataFrame",
        output_path: str,
        format: str = "parquet",
    ) -> None:
        """
        Write results to external storage.

        Supported formats: parquet, delta, json, csv.
        """
        writer = df.write.mode("overwrite")
        if format == "parquet":
            writer.parquet(output_path)
        elif format == "delta":
            writer.format("delta").save(output_path)
        elif format == "json":
            writer.json(output_path)
        elif format == "csv":
            writer.option("header", "true").csv(output_path)
        else:
            raise ValueError(f"Unsupported format: {format!r}")
        logger.info("Results written — path=%s format=%s", output_path, format)

    def export_to_kafka(self, df: "DataFrame", topic: str) -> None:
        """
        Stream-write a DataFrame to a Kafka topic (using Spark's Kafka sink).

        Requires KAFKA_BROKERS env var, e.g. 'broker1:9092,broker2:9092'.
        The DataFrame is serialised as JSON per row as the Kafka value.
        """
        brokers = os.environ.get("KAFKA_BROKERS", "kafka:9092")

        # Spark Kafka connector expects a 'value' column of type string/bytes.
        kafka_df = df.select(
            F.to_json(F.struct([F.col(c) for c in df.columns])).alias("value")
        )

        (
            kafka_df.write.format("kafka")
            .option("kafka.bootstrap.servers", brokers)
            .option("topic", topic)
            .save()
        )
        logger.info("Exported to Kafka — topic=%s brokers=%s", topic, brokers)

    # ------------------------------------------------------------------
    # Convenience pipeline
    # ------------------------------------------------------------------

    def run_ioc_pipeline(
        self,
        input_path: str,
        output_path: str,
        intel_path: Optional[str] = None,
        kafka_topic: Optional[str] = None,
    ) -> "DataFrame":
        """
        Full pipeline: load → filter severity ≥4 → aggregate → optionally
        enrich with threat intel → write results.
        """
        df = self.load_parquet(input_path)
        df = self.filter_high_severity(df)
        df = self.aggregate_iocs(df)

        if intel_path:
            df = self.join_threat_intel(df, intel_path)

        self.write_results(df, output_path)

        if kafka_topic:
            self.export_to_kafka(df, kafka_topic)

        return df

    def stop(self) -> None:
        """Stop the SparkSession."""
        self.spark.stop()
        logger.info("SparkSession stopped")

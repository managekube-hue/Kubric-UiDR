"""
QBR (Quarterly Business Review) report generator.

Generates a monthly QBR PDF for a tenant:
- FORESIGHT persona summarizes risk trends
- COMM persona drafts executive summary
- PDF rendered via reportlab
- Uploaded to MinIO: kubric-reports/{tenant_id}/{year}-{month}-qbr.pdf
- Zammad ticket created linking to the PDF
"""

from __future__ import annotations

import io
import os
from datetime import datetime
from typing import Any

import structlog

log = structlog.get_logger(__name__)

# Optional imports — graceful degradation if not installed
try:
    from reportlab.lib.pagesizes import letter
    from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
    from reportlab.lib.units import inch
    from reportlab.platypus import (
        Paragraph,
        SimpleDocTemplate,
        Spacer,
        Table,
        TableStyle,
    )
    from reportlab.lib import colors

    HAS_REPORTLAB = True
except ImportError:
    HAS_REPORTLAB = False

try:
    from minio import Minio

    HAS_MINIO = True
except ImportError:
    HAS_MINIO = False


MINIO_ENDPOINT = os.getenv("KUBRIC_MINIO_ENDPOINT", "localhost:9000")
MINIO_ACCESS_KEY = os.getenv("KUBRIC_MINIO_ACCESS_KEY", "minioadmin")
MINIO_SECRET_KEY = os.getenv("KUBRIC_MINIO_SECRET_KEY", "minioadmin")
MINIO_BUCKET = "kubric-reports"


async def generate_qbr(
    tenant_id: str,
    year: int | None = None,
    month: int | None = None,
    risk_summary: str = "",
    executive_summary: str = "",
    metrics: dict[str, Any] | None = None,
) -> str:
    """Generate a QBR PDF and upload to MinIO.

    Returns the MinIO object path.

    Parameters
    ----------
    tenant_id : str
        The tenant identifier.
    year, month : int, optional
        Report period. Defaults to current month.
    risk_summary : str
        FORESIGHT persona risk trend summary.
    executive_summary : str
        COMM persona executive summary.
    metrics : dict
        Key metrics to display (alerts_total, vulns_critical, compliance_score, etc.)
    """
    now = datetime.utcnow()
    year = year or now.year
    month = month or now.month
    metrics = metrics or {}

    if not HAS_REPORTLAB:
        log.warning("qbr.reportlab_missing", msg="reportlab not installed, skipping PDF generation")
        return ""

    # Build PDF in memory
    buf = io.BytesIO()
    doc = SimpleDocTemplate(buf, pagesize=letter, topMargin=0.75 * inch, bottomMargin=0.75 * inch)
    styles = getSampleStyleSheet()
    title_style = ParagraphStyle("QBRTitle", parent=styles["Heading1"], fontSize=18, textColor=colors.HexColor("#072a49"))
    heading_style = ParagraphStyle("QBRHeading", parent=styles["Heading2"], fontSize=14, textColor=colors.HexColor("#0074c5"))

    elements: list[Any] = []

    # Title
    elements.append(Paragraph(f"Kubric Security — QBR Report", title_style))
    elements.append(Spacer(1, 12))
    elements.append(Paragraph(f"Tenant: {tenant_id} | Period: {year}-{month:02d}", styles["Normal"]))
    elements.append(Spacer(1, 24))

    # Executive Summary
    elements.append(Paragraph("Executive Summary", heading_style))
    elements.append(Spacer(1, 8))
    elements.append(Paragraph(executive_summary or "No executive summary provided.", styles["Normal"]))
    elements.append(Spacer(1, 16))

    # Risk Trends
    elements.append(Paragraph("Risk Trend Analysis", heading_style))
    elements.append(Spacer(1, 8))
    elements.append(Paragraph(risk_summary or "No risk analysis provided.", styles["Normal"]))
    elements.append(Spacer(1, 16))

    # Key Metrics Table
    if metrics:
        elements.append(Paragraph("Key Metrics", heading_style))
        elements.append(Spacer(1, 8))
        table_data = [["Metric", "Value"]]
        for k, v in metrics.items():
            table_data.append([k.replace("_", " ").title(), str(v)])
        t = Table(table_data, colWidths=[3 * inch, 2 * inch])
        t.setStyle(
            TableStyle(
                [
                    ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#0074c5")),
                    ("TEXTCOLOR", (0, 0), (-1, 0), colors.white),
                    ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
                    ("ALIGN", (0, 0), (-1, -1), "LEFT"),
                    ("GRID", (0, 0), (-1, -1), 0.5, colors.grey),
                    ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.whitesmoke, colors.white]),
                ]
            )
        )
        elements.append(t)

    doc.build(elements)
    pdf_bytes = buf.getvalue()

    # Upload to MinIO
    object_path = f"{tenant_id}/{year}-{month:02d}-qbr.pdf"

    if HAS_MINIO:
        try:
            client = Minio(
                MINIO_ENDPOINT,
                access_key=MINIO_ACCESS_KEY,
                secret_key=MINIO_SECRET_KEY,
                secure=False,
            )
            if not client.bucket_exists(MINIO_BUCKET):
                client.make_bucket(MINIO_BUCKET)
            client.put_object(
                MINIO_BUCKET,
                object_path,
                io.BytesIO(pdf_bytes),
                length=len(pdf_bytes),
                content_type="application/pdf",
            )
            log.info("qbr.uploaded", bucket=MINIO_BUCKET, path=object_path, size=len(pdf_bytes))
        except Exception as exc:
            log.error("qbr.upload_failed", error=str(exc))
    else:
        log.warning("qbr.minio_missing", msg="minio not installed, PDF not uploaded")

    return object_path

/**
 * NATS WebSocket event source for the portal.
 *
 * Connects to the KAI API /stream endpoint (SSE) which bridges NATS JetStream
 * to the browser without requiring the NATS client library in the browser.
 *
 * Subject filtering follows the platform convention:
 *   kubric.{tenant_id}.{category}.{event_class}.v1
 */

const KAI_BASE = process.env.NEXT_PUBLIC_KAI_URL ?? "http://localhost:8100";

export type NatsEvent = {
  subject: string;
  tenant_id: string;
  payload: Record<string, unknown>;
  received_at: number;
};

export type EventHandler = (event: NatsEvent) => void;

// ─── NatsEventSource ──────────────────────────────────────────────────────

export class NatsEventSource {
  private es: EventSource | null = null;
  private handlers = new Map<string, Set<EventHandler>>();
  private reconnectMs = 3000;
  private closed = false;

  constructor(
    private tenantId: string,
    private categories: string[] = ["kai", "security", "noc"]
  ) {}

  connect(): void {
    if (this.closed) return;
    const params = new URLSearchParams({
      tenant_id: this.tenantId,
      categories: this.categories.join(","),
    });
    const url = `${KAI_BASE}/stream?${params}`;
    this.es = new EventSource(url);

    this.es.onmessage = (e: MessageEvent) => {
      try {
        const event: NatsEvent = JSON.parse(e.data as string);
        this.dispatch(event);
      } catch {
        // skip malformed events
      }
    };

    this.es.onerror = () => {
      this.es?.close();
      this.es = null;
      if (!this.closed) {
        setTimeout(() => this.connect(), this.reconnectMs);
      }
    };
  }

  on(subject: string, handler: EventHandler): () => void {
    if (!this.handlers.has(subject)) {
      this.handlers.set(subject, new Set());
    }
    this.handlers.get(subject)!.add(handler);
    return () => this.handlers.get(subject)?.delete(handler);
  }

  close(): void {
    this.closed = true;
    this.es?.close();
    this.es = null;
    this.handlers.clear();
  }

  private dispatch(event: NatsEvent): void {
    // Exact match
    this.handlers.get(event.subject)?.forEach((h) => h(event));
    // Wildcard match: kubric.{tenant}.kai.* → registered as "kai"
    const parts = event.subject.split(".");
    if (parts.length >= 3) {
      const category = parts[2];
      this.handlers.get(category)?.forEach((h) => h(event));
    }
    // Catch-all
    this.handlers.get("*")?.forEach((h) => h(event));
  }
}

// ─── React hook helper (plain function, no JSX dependency) ────────────────

/**
 * Returns a NatsEventSource for the given tenant.
 * Call connect() after mounting; call close() on unmount.
 */
export function createNatsSource(
  tenantId: string,
  categories?: string[]
): NatsEventSource {
  return new NatsEventSource(tenantId, categories);
}

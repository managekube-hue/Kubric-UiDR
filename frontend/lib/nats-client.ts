/**
 * NATS WebSocket client for real-time alert streaming.
 * Connects to NATS via WebSocket and subscribes to tenant-scoped subjects.
 */

import { connect, type NatsConnection, type Subscription, StringCodec } from "nats.ws";

const NATS_WS_URL = process.env.NEXT_PUBLIC_NATS_WS_URL || "ws://localhost:9222";

const sc = StringCodec();

export interface NatsAlert {
  type: string;
  severity: string;
  title: string;
  description: string;
  source: string;
  tenant_id: string;
  timestamp: string;
  metadata?: Record<string, unknown>;
}

let connection: NatsConnection | null = null;

export async function getNatsConnection(): Promise<NatsConnection> {
  if (connection && !connection.isClosed()) {
    return connection;
  }
  connection = await connect({ servers: NATS_WS_URL });
  return connection;
}

export async function subscribeAlerts(
  tenantId: string,
  onAlert: (alert: NatsAlert) => void
): Promise<Subscription> {
  const nc = await getNatsConnection();
  const sub = nc.subscribe(`kubric.${tenantId}.>`);

  (async () => {
    for await (const msg of sub) {
      try {
        const data = JSON.parse(sc.decode(msg.data)) as NatsAlert;
        data.tenant_id = tenantId;
        onAlert(data);
      } catch {
        // skip malformed messages
      }
    }
  })();

  return sub;
}

export async function closeNats(): Promise<void> {
  if (connection) {
    await connection.drain();
    connection = null;
  }
}

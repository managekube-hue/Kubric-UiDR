import { NextRequest, NextResponse } from "next/server";
import Stripe from "stripe";

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY ?? "", {
  apiVersion: "2024-12-18.acacia",
});
const webhookSecret = process.env.STRIPE_WEBHOOK_SECRET ?? "";

export async function POST(req: NextRequest) {
  const body = await req.text();
  const sig = req.headers.get("stripe-signature");

  if (!sig) {
    return NextResponse.json({ error: "Missing signature" }, { status: 400 });
  }

  let event: Stripe.Event;
  try {
    event = stripe.webhooks.constructEvent(body, sig, webhookSecret);
  } catch (err) {
    const message = err instanceof Error ? err.message : "Invalid signature";
    return NextResponse.json({ error: message }, { status: 400 });
  }

  switch (event.type) {
    case "customer.subscription.created": {
      const subscription = event.data.object as Stripe.Subscription;
      // Forward to K-SVC to update tenant subscription tier
      await fetch(`${process.env.NEXT_PUBLIC_API_BASE}/v1/billing/webhook`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          event_type: "subscription.created",
          customer_id: subscription.customer,
          subscription_id: subscription.id,
          status: subscription.status,
        }),
      }).catch(() => {});
      break;
    }
    case "customer.subscription.deleted": {
      const subscription = event.data.object as Stripe.Subscription;
      await fetch(`${process.env.NEXT_PUBLIC_API_BASE}/v1/billing/webhook`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          event_type: "subscription.deleted",
          customer_id: subscription.customer,
          subscription_id: subscription.id,
        }),
      }).catch(() => {});
      break;
    }
    case "invoice.paid": {
      const invoice = event.data.object as Stripe.Invoice;
      await fetch(`${process.env.NEXT_PUBLIC_API_BASE}/v1/billing/webhook`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          event_type: "invoice.paid",
          customer_id: invoice.customer,
          invoice_id: invoice.id,
          amount_paid: invoice.amount_paid,
        }),
      }).catch(() => {});
      break;
    }
  }

  return NextResponse.json({ received: true });
}

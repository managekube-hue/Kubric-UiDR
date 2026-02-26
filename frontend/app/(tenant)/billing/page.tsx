"use client";

import { useSession } from "next-auth/react";
import { useEffect, useState } from "react";
import {
  Card,
  Title,
  Text,
  Metric,
  Flex,
  Badge,
  Grid,
} from "@tremor/react";
import { CreditCard, ExternalLink, ArrowUpRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { getBillingUsage, getBillingPortalUrl, type BillingUsage } from "@/lib/api-client";

export default function BillingPage() {
  const { data: session } = useSession();
  const [usage, setUsage] = useState<BillingUsage | null>(null);
  const [loading, setLoading] = useState(true);
  const [portalLoading, setPortalLoading] = useState(false);

  useEffect(() => {
    if (!session?.accessToken) return;
    setLoading(true);
    getBillingUsage({ token: session.accessToken, tenantId: session.tenantId })
      .then(setUsage)
      .catch(() => setUsage(null))
      .finally(() => setLoading(false));
  }, [session?.accessToken, session?.tenantId]);

  async function openPortal() {
    if (!session?.accessToken) return;
    setPortalLoading(true);
    try {
      const { url } = await getBillingPortalUrl({
        token: session.accessToken,
        tenantId: session.tenantId,
      });
      window.open(url, "_blank");
    } catch {
      // Stripe portal unavailable
    } finally {
      setPortalLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <Flex>
        <div>
          <Title>Billing & Usage</Title>
          <Text>Subscription management and usage metering</Text>
        </div>
        <Button onClick={openPortal} disabled={portalLoading} variant="outline">
          <ExternalLink className="h-4 w-4 mr-2" />
          {portalLoading ? "Loading..." : "Stripe Portal"}
        </Button>
      </Flex>

      {loading ? (
        <Text>Loading billing data...</Text>
      ) : !usage ? (
        <Card>
          <Text>No billing data available. Contact support to set up your subscription.</Text>
        </Card>
      ) : (
        <>
          <Grid numItemsMd={3} className="gap-4">
            <Card decoration="top" decorationColor="blue">
              <Flex alignItems="start">
                <div>
                  <Text>Current Period</Text>
                  <Metric>{usage.period}</Metric>
                </div>
                <CreditCard className="h-6 w-6 text-blue-500" />
              </Flex>
            </Card>
            <Card decoration="top" decorationColor="emerald">
              <Flex alignItems="start">
                <div>
                  <Text>Events Processed</Text>
                  <Metric>{usage.events_count.toLocaleString()}</Metric>
                </div>
                <ArrowUpRight className="h-6 w-6 text-emerald-500" />
              </Flex>
            </Card>
            <Card decoration="top" decorationColor="violet">
              <Flex alignItems="start">
                <div>
                  <Text>Active Agents</Text>
                  <Metric>{usage.agents_count}</Metric>
                </div>
              </Flex>
            </Card>
          </Grid>

          <Card>
            <Flex>
              <div>
                <Text>Estimated Amount</Text>
                <Metric>
                  ${usage.total_amount.toLocaleString(undefined, {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 2,
                  })}
                </Metric>
              </div>
              <Badge color="blue">Metered billing</Badge>
            </Flex>
            <Text className="mt-4 text-xs text-gray-500">
              Usage is metered per event processed and per active agent. View
              detailed invoices and payment methods in the Stripe portal.
            </Text>
          </Card>
        </>
      )}
    </div>
  );
}

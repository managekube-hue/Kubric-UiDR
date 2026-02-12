import React from 'react';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';

export default function Compliance() {
  return (
    <Layout title="Compliance" description="Kubric compliance and audit resources">
      <main className="container margin-vert--lg">
        <h1>Compliance & Audit</h1>
        <p>Resources for SOC2, ISO and other compliance artifacts.</p>
        <ul>
          <li><Link to="/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/">SOC2 Evidence Vault</Link></li>
          <li><Link to="/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-05_AUDIT_READINESS/">Audit Readiness</Link></li>
          <li><Link to="/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC-003_LICENSE">License & NOTICE</Link></li>
        </ul>
      </main>
    </Layout>
  );
}

import React from 'react';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';

export default function Modules() {
  return (
    <Layout title="Modules" description="Kubric modules overview">
      <main className="container margin-vert--lg">
        <h1>Kubric Modules</h1>
        <p>Quick links to major documentation modules.</p>
        <ul>
          <li><Link to="/docs/K-CORE-01_INFRASTRUCTURE/">Core Infrastructure</Link></li>
          <li><Link to="/docs/K-XRO-02_SUPER_AGENT/">XRO Super Agent</Link></li>
          <li><Link to="/docs/K-KAI-03_ORCHESTRATION/">KAI Orchestration</Link></li>
          <li><Link to="/docs/K-SOC-04_SECURITY/">SOC Security</Link></li>
          <li><Link to="/docs/K-NOC-05_OPERATIONS/">NOC Operations</Link></li>
          <li><Link to="/docs/K-PSA-06_BUSINESS/">PSA Business</Link></li>
          <li><Link to="/docs/K-GRC-07_COMPLIANCE/">GRC Compliance</Link></li>
          <li><Link to="/docs/K-DEV-08_DEVELOPMENT/">Development</Link></li>
          <li><Link to="/docs/K-API-09_API_REFERENCE/">API Reference</Link></li>
          <li><Link to="/docs/K-ITIL-10_ITIL_MATRIX/">ITIL Map</Link></li>
        </ul>
      </main>
    </Layout>
  );
}

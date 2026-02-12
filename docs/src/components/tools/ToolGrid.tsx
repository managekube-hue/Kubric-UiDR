import React from 'react';
import styles from './ToolGrid.module.css';

const tools = [
  {
    name: 'Slack',
    description: 'Real-time communication, community support, engineering discussions',
    purpose: 'Communication',
    audience: 'All contributors',
    channel: '#contributors, #engineering',
  },
  {
    name: 'Linear',
    description: 'Issue tracking, sprint planning, roadmap progression',
    purpose: 'Project Management',
    audience: 'Engineering team',
    channel: 'Project: Kubric UIDR',
  },
  {
    name: 'GitHub',
    description: 'Source control, pull requests, code review, CI/CD',
    purpose: 'Development',
    audience: 'All contributors',
    channel: 'kubric/platform',
  },
  {
    name: 'Notion',
    description: 'Requirements, specifications, architectural decisions',
    purpose: 'Documentation',
    audience: 'All contributors',
    channel: 'Kubric Workspace',
  },
  {
    name: 'Zoom',
    description: 'Weekly standups, architecture reviews, pair programming',
    purpose: 'Meetings',
    audience: 'Engineering team',
    channel: 'Weekly · Wed 15:00 UTC',
  },
  {
    name: 'Asana',
    description: 'Contributor onboarding, task tracking, project milestones',
    purpose: 'Project Management',
    audience: 'Core team',
    channel: 'Kubric · Platform',
  },
  {
    name: 'Figma',
    description: 'UI/UX design, component library, customer portal',
    purpose: 'Design',
    audience: 'Design + Engineering',
    channel: 'Kubric Design System',
  },
  {
    name: '1Password',
    description: 'Shared credentials, environment variables, secrets',
    purpose: 'Security',
    audience: 'Core team',
    channel: 'Vault: Kubric Engineering',
  },
];

export function ToolGrid() {
  return (
    <section className={styles.section}>
      <div className={styles.container}>
        <div className={styles.header}>
          <h2 className={styles.title}>Our toolchain</h2>
          <p className={styles.subtitle}>
            Everything you need to contribute effectively. No silos, no friction.
          </p>
        </div>
        <div className={styles.grid}>
          {tools.map((tool) => (
            <div key={tool.name} className={styles.card}>
              <div className={styles.cardHeader}>
                <h3 className={styles.toolName}>{tool.name}</h3>
                <span className={styles.purpose}>{tool.purpose}</span>
              </div>
              <p className={styles.description}>{tool.description}</p>
              <div className={styles.metadata}>
                <div className={styles.metadataItem}>
                  <span className={styles.metadataLabel}>Audience:</span>
                  <span className={styles.metadataValue}>{tool.audience}</span>
                </div>
                <div className={styles.metadataItem}>
                  <span className={styles.metadataLabel}>Channel:</span>
                  <span className={styles.metadataValue}>{tool.channel}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

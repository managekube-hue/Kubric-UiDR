import React from 'react';
import Link from '@docusaurus/Link';
import {
  documentationMap,
  getGithubUrl,
  getNotionUrl,
} from '@site/src/lib/docs/mapping';
import styles from './PlatformModuleGrid.module.css';

const primaryModuleCodes = [
  'K-SOC-DET',
  'K-NOC-CM',
  'K-GRC-OSCAL',
  'K-PSA-ITSM',
  'K-KAI-CP',
  'K-XRO-CS',
];

const moduleCards = primaryModuleCodes
  .map((code) => documentationMap.find((entry) => entry.moduleCode === code))
  .filter((entry): entry is NonNullable<typeof entry> => Boolean(entry));

export default function PlatformModuleGrid() {
  return (
    <div className="enterpriseCardGrid">
      {moduleCards.map((module) => (
        <article key={module.moduleCode} className="enterpriseCard">
          <h3>{module.moduleName}</h3>
          <p>{module.moduleCode}</p>
          <div className={styles.docLinks}>
            <Link className="enterpriseCard__cta" to={`/docs/${module.slug.replace(/\/index$/, '')}`}>
              Technical documentation
            </Link>
            <a className={styles.textLink} href={getNotionUrl(module.notionId)} target="_blank" rel="noopener noreferrer">
              Stakeholder context ↗
            </a>
            <a className={styles.textLink} href={getGithubUrl(module.githubPath)} target="_blank" rel="noopener noreferrer">
              View source ↗
            </a>
          </div>
        </article>
      ))}
    </div>
  );
}

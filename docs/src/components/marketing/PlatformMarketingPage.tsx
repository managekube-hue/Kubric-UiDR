import React, {type ReactNode} from 'react';
import Link from '@docusaurus/Link';
import type {PlatformPageContent} from '@site/src/lib/marketing/platformContent';
import MarketingBreadcrumbs from '@site/src/components/marketing/MarketingBreadcrumbs';

interface Props {
  page: PlatformPageContent;
}

export default function PlatformMarketingPage({page}: Props): ReactNode {
  const breadcrumbs = [
    {label: 'Home', href: '/'},
    {label: 'Platform', href: '/platform'},
    ...(page.key === 'overview' ? [] : [{label: page.title.replace('Platform ', ''), href: `/platform/${page.key}`}]),
  ];

  return (
    <div className="enterprisePage">
      <section className="enterpriseHero">
        <div className="enterpriseHero__inner">
          <MarketingBreadcrumbs items={breadcrumbs} />
          <p className="enterpriseHero__eyebrow">{page.eyebrow}</p>
          <h1>{page.headline}</h1>
          <p>{page.subheadline}</p>
          <div className="enterpriseHero__actions">
            <Link className="enterpriseButton enterpriseButton--primary" to={page.ctaHref}>
              {page.ctaLabel}
            </Link>
            {page.secondaryLabel && page.secondaryHref ? (
              <Link className="enterpriseButton" to={page.secondaryHref}>
                {page.secondaryLabel}
              </Link>
            ) : null}
          </div>
          <p className="enterpriseHero__proof">Hero, headlines, icons, and visual bands are code-managed while body content follows the Notion page template model.</p>
          <div className="enterpriseHero__band" aria-hidden="true">
            <img src="/img/kubric-hero-band.svg" alt="" />
          </div>
        </div>
      </section>

      <main className="enterpriseHome">
        <section>
          <h2>Kube Cards</h2>
          <p className="moduleBandStatement">{page.bandStatement}</p>
          <div className="enterpriseCardGrid enterpriseCardGrid--raised">
            {page.features.map((feature) => (
              <article className="enterpriseCard enterpriseCard--raised" key={feature.href}>
                <h3>{feature.title}</h3>
                <p>{feature.description}</p>
                <Link className="enterpriseCard__cta" to={feature.href}>
                  Open section
                </Link>
              </article>
            ))}
          </div>
        </section>

        <section className="enterpriseWhy moduleWhySection">
          <h2>Why this module matters</h2>
          <p>
            {page.moduleWhy}
          </p>
        </section>

        <section className="enterpriseWhy roadmapSection">
          <h2>Roadmap to production</h2>
          <ol className="roadmapList">
            {page.roadmap.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>
        </section>

        <section className="enterpriseWhy contributorCta">
          <h2>Contributors can accelerate this module</h2>
          <p>Join implementation, testing, and operational hardening tracks for this module.</p>
          <Link className="enterpriseButton enterpriseButton--primary" to="/contributors">
            Become a Contributor
          </Link>
        </section>
      </main>
    </div>
  );
}

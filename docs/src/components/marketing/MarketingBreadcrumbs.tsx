import React from 'react';
import Link from '@docusaurus/Link';

interface BreadcrumbItem {
  label: string;
  href: string;
}

interface Props {
  items: BreadcrumbItem[];
}

export default function MarketingBreadcrumbs({items}: Props): JSX.Element {
  return (
    <nav className="marketingBreadcrumbs" aria-label="Breadcrumbs">
      {items.map((item, index) => (
        <span key={item.href} className="marketingBreadcrumbs__item">
          {index > 0 ? <span className="marketingBreadcrumbs__sep">/</span> : null}
          <Link to={item.href}>{item.label}</Link>
        </span>
      ))}
    </nav>
  );
}

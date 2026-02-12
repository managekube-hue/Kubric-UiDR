import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';

const config: Config = {
  title: 'Kubric Enterprise Platform',
  tagline: 'Enterprise Security Operations & Orchestration',
  favicon: 'img/favicon.ico',
  url: 'https://kubric-platform.vercel.app',
  baseUrl: '/',
  organizationName: 'kubric',
  projectName: 'kubric-platform',
  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',
  trailingSlash: false,
  
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.js',
          editUrl: 'https://github.com/kubric/kubric-uidr/tree/main/docs/',
          routeBasePath: '/docs',
          include: ['**/*.{md,mdx}'],
          exclude: ['**/README.md', '**/node_modules/**'],
          showLastUpdateTime: true,
          showLastUpdateAuthor: true,
          versions: {
            current: {
              label: 'Next',
              path: 'next',
              banner: 'unreleased',
            },
          },
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      } as any,
    ],
  ],

  themeConfig: {
    image: 'img/kubric-og.png',
    metadata: [
      {
        name: 'description',
        content: 'Kubric: Enterprise Security Operations & Orchestration Platform with autonomous agents, real-time threat intelligence, and ITIL compliance',
      },
      {
        name: 'keywords',
        content: 'SOC, NOC, GRC, security operations, threat intelligence, automation, orchestration, ITIL, compliance',
      },
      {
        property: 'og:type',
        content: 'website',
      },
      {
        property: 'og:title',
        content: 'Kubric - Enterprise Security Platform',
      },
      {
        name: 'twitter:card',
        content: 'summary_large_image',
      },
      {
        httpEquiv: 'x-ua-compatible',
        content: 'IE=edge',
      },
    ],
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    announcementBar: {
      id: 'kubric-announcement',
      content: '🚀 Enterprise-Grade Security Operations Platform - <strong>Version 1.0</strong>',
      backgroundColor: '#FF8533',
      textColor: '#FFFFFF',
      isCloseable: true,
    },
    navbar: {
      title: 'Kubric',
      hideOnScroll: false,
      style: 'dark',
      logo: {
        alt: 'Kubric Logo',
        src: 'img/kubric-logo.svg',
        srcDark: 'img/kubric-logo-dark.svg',
        height: 32,
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'tutorialSidebar',
          position: 'left',
          label: '📚 Documentation',
        },
        {
          type: 'docsVersionDropdown',
          position: 'right',
          dropdownActiveClassDisabled: true,
        },
        {
          href: 'https://github.com/kubric/kubric-uidr',
          position: 'right',
          className: 'header-github-link',
          'aria-label': 'GitHub repository',
        },
      ],
    },
    footer: {
      style: 'dark',
      logo: {
        alt: 'Kubric',
        src: 'img/kubric-logo.svg',
        height: 50,
        width: 50,
      },
      links: [
        {
          title: 'Core Modules',
          items: [
            {
              label: '🏗️ Infrastructure',
              to: '/docs/K-CORE-01_INFRASTRUCTURE',
            },
            {
              label: '🤖 Orchestration',
              to: '/docs/K-KAI-03_ORCHESTRATION',
            },
            {
              label: '🛡️ Security Operations',
              to: '/docs/K-SOC-04_SECURITY',
            },
            {
              label: '📊 Operations',
              to: '/docs/K-NOC-05_OPERATIONS',
            },
          ],
        },
        {
          title: 'Enterprise',
          items: [
            {
              label: '💼 Business & Billing',
              to: '/docs/K-PSA-06_BUSINESS',
            },
            {
              label: '📋 Compliance & GRC',
              to: '/docs/K-GRC-07_COMPLIANCE',
            },
            {
              label: '🔧 Development',
              to: '/docs/K-DEV-08_DEVELOPMENT',
            },
          ],
        },
        {
          title: 'Reference',
          items: [
            {
              label: '📡 API Reference',
              to: '/docs/K-API-09_API_REFERENCE',
            },
            {
              label: '🚀 Super Agent',
              to: '/docs/K-XRO-02_SUPER_AGENT',
            },
            {
              label: '📚 ITIL Framework',
              to: '/docs/K-ITIL-10_ITIL_MATRIX',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/kubric/kubric-uidr',
            },
            {
              label: 'Issues',
              href: 'https://github.com/kubric/kubric-uidr/issues',
            },
            {
              label: 'Discussions',
              href: 'https://github.com/kubric/kubric-uidr/discussions',
            },
          ],
        },
      ],
      copyright: `
        <div style="text-align: center; color: #FFFFFF;">
          <p style="margin: 0 0 8px 0; font-weight: 600;">© ${new Date().getFullYear()} Kubric Platform. All rights reserved.</p>
          <p style="margin: 0; color: #CCCCCC; font-size: 0.9em;">Enterprise Security Operations & Orchestration</p>
          <p style="margin: 8px 0 0 0; color: #999999; font-size: 0.85em;">
            Built with Docusaurus | Deployed on Vercel
          </p>
        </div>
      `,
    },
    prism: {
      theme: prismThemes.oneDark,
      darkTheme: prismThemes.oneDark,
      additionalLanguages: ['bash', 'python', 'typescript', 'yaml', 'json', 'sql', 'rust', 'go', 'java', 'csharp', 'javascript', 'toml'],
      magicComments: [
        {
          className: 'theme-code-block-highlighted-line',
          line: 'highlight-next-line',
          block: {start: 'highlight-start', end: 'highlight-end'},
        },
        {
          className: 'code-block-error-line',
          line: 'error-line',
        },
      ],
    },
    algolia: {
      appId: process.env.ALGOLIA_APP_ID || 'PLACEHOLDER',
      apiKey: process.env.ALGOLIA_SEARCH_API_KEY || 'PLACEHOLDER',
      indexName: 'kubric-docs',
      contextualSearch: true,
      searchPagePath: 'search',
    },
    tableOfContents: {
      minHeadingLevel: 2,
      maxHeadingLevel: 4,
    },
  } as any,

  plugins: [
    // Core Docusaurus plugins
    '@docusaurus/plugin-content-docs',
    '@docusaurus/plugin-content-pages',
    
    // SEO & Discovery
    [
      '@docusaurus/plugin-sitemap',
      {
        changefreq: 'weekly',
        priority: 0.5,
        ignorePatterns: ['/tags/**', '/docs/**/**/index.md'],
        trailingSlash: false,
      },
    ],

    // Analytics & Monitoring (uses environment variables)
    [
      '@docusaurus/plugin-google-gtag',
      {
        trackingID: process.env.GOOGLE_GA_ID || 'UA-XXXXXXXXX-X',
        anonymizeIP: true,
      },
    ],

    // Performance & PWA
    [
      '@docusaurus/plugin-pwa',
      {
        offlineModeActivationStrategies: ['appInstalled', 'standalone', 'queryString'],
        pwaHead: [
          {
            tagName: 'link',
            rel: 'icon',
            href: '/img/favicon.ico',
          },
          {
            tagName: 'link',
            rel: 'manifest',
            href: '/manifest.json',
          },
          {
            tagName: 'meta',
            name: 'theme-color',
            content: '#FF8533',
          },
          {
            tagName: 'meta',
            name: 'apple-mobile-web-app-capable',
            content: 'yes',
          },
          {
            tagName: 'meta',
            name: 'apple-mobile-web-app-status-bar-style',
            content: '#000000',
          },
          {
            tagName: 'link',
            rel: 'apple-touch-icon',
            href: '/img/kubric-logo.png',
          },
        ],
        dest: '.docusaurus/pwa',
        filename: 'sw.js',
        progressBar: true,
        offlinePages: [],
        workboxOptions: {
          skipWaiting: true,
          clientsClaim: true,
          maximumFileSizeToCacheInBytes: 5 * 1024 * 1024,
        },
      },
    ],

    // Image Optimization
    [
      '@docusaurus/plugin-ideal-image',
      {
        quality: 85,
        max: 2000,
        min: 640,
        steps: 2,
        disableInDev: false,
      },
    ],

    // Client-side Redirects (for breaking changes in docs)
    [
      '@docusaurus/plugin-client-redirects',
      {
        redirects: [
          {
            from: ['/docs/intro', '/docs/introduction'],
            to: '/docs/K-CORE-01_INFRASTRUCTURE',
          },
          {
            from: '/docs/api',
            to: '/docs/K-API-09_API_REFERENCE',
          },
        ],
      },
    ],

    // Local Search (fallback if Algolia unavailable)
    [
      '@cmfcmf/docusaurus-search-local',
      {
        language: ['en'],
        hashed: true,
        docsRouteBasePath: '/docs',
        highlightSearchTermsOnTargetPage: true,
        explicitSearchResultPath: true,
        indexBlog: false,
        indexPages: true,
        maxSearchResults: 8,
      },
    ],
  ],

  themes: [
    // Mermaid diagrams for architecture & workflows
    '@docusaurus/theme-mermaid',
    
    // Live code editing in docs
    '@docusaurus/theme-live-codeblock',
    
    // Algolia search theme
    '@docusaurus/theme-search-algolia',
  ],

  markdown: {
    mermaid: true,
    mdx1Compat: {
      comments: false,
      admonitions: true,
      headingIds: true,
    },
  },

  scripts: [
    // Enhanced Mermaid with better rendering
    {
      src: 'https://cdn.jsdelivr.net/npm/mermaid@latest/dist/mermaid.min.js',
      async: true,
    },
    // Structured data for SEO
    {
      src: '/js/structured-data.js',
      async: true,
    },
  ],

  headTags: [
    {
      tagName: 'link',
      attributes: {
        rel: 'preconnect',
        href: 'https://cdn.jsdelivr.net',
      },
    },
  ],

  customFields: {
    // Enterprise metadata
    organization: 'Kubric',
    productName: 'Kubric Enterprise Platform',
    productVersion: '1.0.0',
    supportEmail: 'support@kubric.io',
    salesEmail: 'sales@kubric.io',
    documentationVersion: '1.0',
    lastUpdated: new Date().toISOString(),
  },
};

export default config;

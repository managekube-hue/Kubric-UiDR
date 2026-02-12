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
        },
        gtag: process.env.GOOGLE_GA_ID
          ? {
              trackingID: process.env.GOOGLE_GA_ID,
              anonymizeIP: true,
            }
          : undefined,
        sitemap: {
          changefreq: 'weekly',
          priority: 0.5,
          filename: 'sitemap.xml',
        },
        blog: false,
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
        content: 'Kubric: Enterprise Security Operations & Orchestration Platform',
      },
      {
        name: 'keywords',
        content: 'SOC, NOC, GRC, security operations, automation, orchestration',
      },
    ],
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    announcementBar: {
      id: 'kubric-announcement',
      content: '🚀 Enterprise-Grade Security Operations Platform - Version 1.0',
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
          ],
        },
      ],
      copyright: `<div style="text-align: center; color: #FFFFFF;">
        <p style="margin: 0 0 8px 0; font-weight: 600;">© ${new Date().getFullYear()} Kubric Platform</p>
        <p style="margin: 0; font-size: 0.9em;">Enterprise Security Operations & Orchestration</p>
      </div>`,
    },
    prism: {
      theme: prismThemes.oneDark,
      darkTheme: prismThemes.oneDark,
      additionalLanguages: ['bash', 'python', 'typescript', 'yaml', 'json', 'sql', 'rust', 'go', 'java', 'javascript', 'jsx'],
    },
    algolia: {
      appId: process.env.ALGOLIA_APP_ID || 'PLACEHOLDER',
      apiKey: process.env.ALGOLIA_SEARCH_API_KEY || 'PLACEHOLDER',
      indexName: 'kubric-docs',
    },
  } as any,

  plugins: [
    [
      '@docusaurus/plugin-ideal-image',
      {
        quality: 85,
        max: 1920,
        min: 640,
        steps: 2,
        disableInDev: false,
      },
    ],
    [
      '@docusaurus/plugin-pwa',
      {
        offlineModeActivationStrategies: ['appInstalled', 'standalone', 'queryString'],
        pwaHead: [
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
        ],
      },
    ],
  ],

  themes: [
    '@docusaurus/theme-mermaid',
    '@docusaurus/theme-live-codeblock',
  ],

  markdown: {
    mermaid: true,
  },

  scripts: [
    {
      src: 'https://cdn.jsdelivr.net/npm/mermaid@latest/dist/mermaid.min.js',
      async: true,
    },
  ],

  customFields: {
    organization: 'Kubric',
    productVersion: '1.0.0',
  },
};

export default config;

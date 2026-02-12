import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';

const config: Config = {
  title: 'Kubric Platform',
  tagline: 'Enterprise Security Operations & Orchestration',
  favicon: 'img/favicon.ico',
  url: 'https://kubric-platform.vercel.app',
  baseUrl: '/',
  organizationName: 'kubric',
  projectName: 'kubric-platform',
  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',
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
          editUrl: 'https://github.com/kubric/kubric-uidr/edit/main/docs/',
          admonitions: {
            keywords: ['note', 'tip', 'info', 'caution', 'danger', 'success', 'warning'],
            extendDefaults: true,
          },
        },
        pages: {
          path: 'src/pages',
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      } as any,
    ],
  ],

  themeConfig: {
    image: 'img/kubric-og.png',
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Kubric Platform',
      logo: {
        alt: 'Kubric Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'tutorialSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          href: 'https://github.com/kubric/kubric-uidr',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Getting Started',
              to: '/docs/intro',
            },
            {
              label: 'Architecture',
              to: '/docs/architecture',
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
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Kubric Platform. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.oneDark,
      darkTheme: prismThemes.oneDark,
      additionalLanguages: ['bash', 'python', 'typescript', 'yaml', 'json', 'sql'],
    },
  } as any,

  plugins: [
    // Local search with Chinese/English support
    [
      '@easyops-cn/docusaurus-search-local',
      {
        hashed: true,
        language: ['en', 'zh'],
        docsRouteBasePath: '/docs',
        highlightSearchTermsOnTargetPage: true,
      },
    ],

    // OpenAPI documentation
    [
      'docusaurus-plugin-openapi-docs',
      {
        id: 'openapi',
        docsPluginId: 'classic',
        config: {
          api: {
            specPath: 'openapi/spec.yaml',
            outputDir: 'docs/api/openapi',
            downloadUrl: 'https://raw.githubusercontent.com/kubric/kubric-uidr/main/openapi/spec.yaml',
          },
        },
      },
    ],

    // Redoc for alternative OpenAPI rendering
    [
      'redocusaurus',
      {
        specs: [
          {
            url: 'https://raw.githubusercontent.com/kubric/kubric-uidr/main/openapi/spec.yaml',
            route: '/api/redoc',
          },
        ],
        theme: {
          primaryColor: '#FF8533',
          textColor: '#FFFFFF',
          backgroundColor: '#1a1a1a',
        },
      },
    ],

    // Structured data for SEO
    [
      '@coffeecup_tech/docusaurus-plugin-structured-data',
      {
        schema: {
          '@context': 'https://schema.org/',
          '@type': 'WebSite',
          name: 'Kubric Platform',
          description: 'Enterprise Security Operations & Orchestration',
          url: 'https://kubric-platform.vercel.app',
          author: {
            '@type': 'Organization',
            name: 'Kubric',
          },
        },
      },
    ],

    // Sitemap generation
    [
      '@docusaurus/plugin-sitemap',
      {
        changefreq: 'weekly',
        priority: 0.5,
        trailingSlash: false,
      },
    ],

    // Progressive Web App support
    [
      '@docusaurus/plugin-pwa',
      {
        debug: process.env.NODE_ENV !== 'production',
        offlineModeActivationStrategies: ['appInstalled', 'queryString'],
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
        ],
      },
    ],

    // Image optimization
    [
      '@docusaurus/plugin-ideal-image',
      {
        quality: 70,
        max: 1030,
        min: 640,
        steps: 2,
        disableInDev: false,
      },
    ],

    // Client-side redirects
    [
      '@docusaurus/plugin-client-redirects',
      {
        redirects: [
          {
            from: '/docs/old-page',
            to: '/docs/new-page',
          },
        ],
        createRedirects(existingPath) {
          if (existingPath.includes('/api/')) {
            return [
              existingPath.replace('/api/', '/rest-api/'),
            ];
          }
          return undefined;
        },
      },
    ],
  ],

  themes: [
    // Mermaid diagram support
    '@docusaurus/theme-mermaid',
    // Live code block editing
    '@docusaurus/theme-live-codeblock',
    // OpenAPI docs theme
    'docusaurus-theme-openapi-docs',
  ],

  markdown: {
    mermaid: true,
  },

  scripts: [
    {
      src: 'https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js',
      async: true,
    },
  ],
};

export default config;

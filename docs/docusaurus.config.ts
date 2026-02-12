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
    ['@easyops-cn/docusaurus-search-local', {
      hashed: true,
      language: ['en', 'zh'],
      docsRouteBasePath: '/docs',
      highlightSearchTermsOnTargetPage: true,
    }],

    ['docusaurus-plugin-openapi-docs', {
      id: 'openapi',
      docsPluginId: 'classic',
      config: {
        kubric_api: {
          specPath: 'openapi/spec.yaml',
          outputDir: 'docs/api/generated',
          downloadUrl: 'https://raw.githubusercontent.com/kubric/kubric-uidr/main/openapi/spec.yaml',
        },
      },
    }],

    ['redocusaurus', {
      specs: [
        {
          url: 'https://raw.githubusercontent.com/kubric/kubric-uidr/main/openapi/spec.yaml',
          route: '/api/redoc',
        },
      ],
      theme: {
        primaryColor: '#FF8533',
      },
    }],

    ['@coffeecup_tech/docusaurus-plugin-structured-data', {
      schema: {
        '@context': 'https://schema.org/',
        '@type': 'WebSite',
        name: 'Kubric Platform',
        description: 'Enterprise Security Operations & Orchestration',
        url: 'https://kubric-platform.vercel.app',
      },
    }],

    ['@docusaurus/plugin-sitemap', {
      changefreq: 'weekly',
      priority: 0.5,
      trailingSlash: false,
    }],

    ['@docusaurus/plugin-pwa', {
      debug: false,
      offlineModeActivationStrategies: ['appInstalled', 'queryString'],
      pwaHead: [
        {
          tagName: 'link',
          rel: 'icon',
          href: '/img/favicon.ico',
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
      ],
    }],

    ['@docusaurus/plugin-ideal-image', {
      quality: 70,
      max: 1030,
      min: 640,
      steps: 2,
    }],

    ['@docusaurus/plugin-client-redirects', {
      redirects: [],
    }],
  ],

  themes: [
    '@docusaurus/theme-mermaid',
    '@docusaurus/theme-live-codeblock',
    'docusaurus-theme-openapi-docs',
  ],

  markdown: {
    mermaid: true,
  },
};

export default config;

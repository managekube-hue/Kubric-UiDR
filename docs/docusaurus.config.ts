import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';

const config: Config = {
  title: 'Kubric',
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
    announcementBar: {
      id: 'enterprisegrade',
      content: 'Enterprise-Grade Security Operations Platform',
      backgroundColor: '#FF8533',
      textColor: '#FFFFFF',
      isCloseable: false,
    },
    navbar: {
      title: 'Kubric',
      logo: {
        alt: 'Kubric Logo',
        src: 'img/kubric-logo.svg',
        srcDark: 'img/kubric-logo-dark.svg',
      },
      style: 'dark',
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
      logo: {
        alt: 'Kubric',
        src: 'img/kubric-logo.svg',
        height: 50,
      },
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Core Infrastructure',
              to: '/docs/K-CORE-01_INFRASTRUCTURE',
            },
            {
              label: 'Security Operations',
              to: '/docs/K-SOC-04_SECURITY',
            },
            {
              label: 'API Reference',
              to: '/docs/K-API-09_API_REFERENCE',
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
      copyright: `<div style="color: #FFFFFF;">Copyright © ${new Date().getFullYear()} Kubric Platform. All rights reserved.</div><div style="color: #CCCCCC; font-size: 0.9em; margin-top: 10px;">Enterprise Security Operations & Orchestration</div>`,
    },
    prism: {
      theme: prismThemes.oneDark,
      darkTheme: prismThemes.oneDark,
      additionalLanguages: ['bash', 'python', 'typescript', 'yaml', 'json', 'sql', 'rust', 'go'],
    },
  } as any,

  plugins: [
    '@docusaurus/plugin-sitemap',
    '@docusaurus/plugin-pwa',
    '@docusaurus/plugin-ideal-image',
    '@docusaurus/plugin-client-redirects',
    '@easyops-cn/docusaurus-search-local',
    'docusaurus-plugin-openapi-docs',
    'redocusaurus',
  ],

  themes: [
    '@docusaurus/theme-mermaid',
    '@docusaurus/theme-live-codeblock',
    '@docusaurus/theme-search-algolia',
    'docusaurus-theme-openapi-docs',
  ],

  markdown: {
    mermaid: true,
  },
};

export default config;

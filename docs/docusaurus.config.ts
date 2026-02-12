import { themes as prismThemes } from 'prism-react-renderer';
import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Kubric UIDR',
  tagline: 'Unified SOC • NOC • GRC • PSA • KAI',
  favicon: 'img/favicon.ico',

  url: 'https://docs.kubric.io',
  baseUrl: '/',
  organizationName: 'kubric',
  projectName: 'platform',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'throw',
  onDuplicateRoutes: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/kubric/platform/edit/main/',
          routeBasePath: '/docs',
          path: 'docs',
          include: ['**/*.md', '**/*.mdx'],
          exclude: [
            '**/_*.{md,mdx}',
            '**/README.md',
            '**/node_modules/**',
          ],
          showLastUpdateTime: true,
          showLastUpdateAuthor: true,
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
        sitemap: {
          changefreq: 'daily',
          priority: 0.7,
          filename: 'sitemap.xml',
          ignorePatterns: ['/docs/K-ITIL-10_ITIL_MATRIX/**'],
        },
        gtag: {
          trackingID: 'G-XXXXXXXXXX',
          anonymizeIP: true,
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/kubric-social-card.jpg',
    navbar: {
      title: 'Kubric UIDR',
      logo: {
        alt: 'Kubric Logo',
        src: 'img/logo.svg',
        srcDark: 'img/logo-dark.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'kubricSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          type: 'docsVersionDropdown',
          position: 'left',
          dropdownActiveClassDisabled: true,
          dropdownItemsAfter: [
            {
              type: 'html',
              value: '<hr class="dropdown-separator">',
            },
            {
              type: 'html',
              value: '<span class="dropdown-subtitle">Archived</span>',
            },
            {
              to: '/versions',
              label: 'All versions',
            },
          ],
        },
        {
          href: 'https://github.com/kubric/platform',
          label: 'GitHub',
          position: 'right',
        },
        {
          type: 'search',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Modules',
          items: [
            { label: ' Core Infrastructure', to: '/docs/K-CORE-01_INFRASTRUCTURE/' },
            { label: ' XRO Super Agent', to: '/docs/K-XRO-02_SUPER_AGENT/' },
            { label: ' KAI Orchestration', to: '/docs/K-KAI-03_ORCHESTRATION/' },
            { label: 'SOC Security', to: '/docs/K-SOC-04_SECURITY/' },
            { label: ' NOC Operations', to: '/docs/K-NOC-05_OPERATIONS/' },
            { label: ' PSA Business', to: '/docs/K-PSA-06_BUSINESS/' },
            { label: 'GRC Compliance', to: '/docs/K-GRC-07_COMPLIANCE/' },
            { label: 'Development', to: '/docs/K-DEV-08_DEVELOPMENT/' },
            { label: ' API Reference', to: '/docs/K-API-09_API_REFERENCE/' },
            { label: ' ITIL Map', to: '/docs/K-ITIL-10_ITIL_MATRIX/' },
          ],
        },
        {
          title: 'Compliance',
          items: [
            { label: 'SOC2 Evidence', to: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/' },
            { label: 'ISO 27001', to: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-05_AUDIT_READINESS/K-ITIL-AUD-002_SOC2_ISO_Control_Crosswalk' },
            { label: 'NIST 800-53', to: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-05_AUDIT_READINESS/' },
            { label: 'License', to: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC-003_LICENSE' },
            { label: 'NOTICE', to: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC-004_NOTICE' },
          ],
        },
        {
          title: 'Community',
          items: [
            { label: 'GitHub', href: 'https://github.com/kubric/platform' },
            { label: 'Discord', href: 'https://discord.gg/kubric' },
            { label: 'Twitter', href: 'https://twitter.com/kubric' },
            { label: 'Stack Overflow', href: 'https://stackoverflow.com/questions/tagged/kubric' },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Kubric. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: [
        'rust', 'go', 'python', 'typescript', 'yaml', 'json', 'sql', 
        'bash', 'protobuf', 'toml', 'ini', 'nginx', 'hcl', 'groovy', 
        'java', 'csharp', 'powershell', 'markdown', 'docker', 'diff'
      ],
      magicComments: [
        {
          className: 'theme-code-block-highlighted-line',
          line: 'highlight-next-line',
          block: { start: 'highlight-start', end: 'highlight-end' },
        },
      ],
    },
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    docs: {
      sidebar: {
        hideable: true,
        autoCollapseCategories: true,
      },
    },
    tableOfContents: {
      minHeadingLevel: 2,
      maxHeadingLevel: 5,
    },
    algolia: {
      appId: 'YOUR_APP_ID',
      apiKey: 'YOUR_SEARCH_API_KEY',
      indexName: 'kubric',
      contextualSearch: true,
      searchParameters: {},
      searchPagePath: 'search',
    },
    metadata: [
      { name: 'keywords', content: 'security, soc, noc, grc, psa, xdr, edr, ndr, compliance, itil' },
      { name: 'twitter:card', content: 'summary_large_image' },
      { name: 'twitter:site', content: '@kubric' },
    ],
  } satisfies Preset.ThemeConfig,

  plugins: [
    [
      '@docusaurus/plugin-search-local',
      {
        indexDocs: true,
        indexBlog: false,
        indexPages: true,
        language: ['en'],
        highlightSearchTermsOnTargetPage: true,
        searchResultLimits: 8,
        searchResultContextMaxLength: 50,
        docsDir: ['docs'],
      },
    ],
    '@docusaurus/plugin-sitemap',
    [
      '@docusaurus/plugin-pwa',
      {
        debug: false,
        offlineModeActivationStrategies: [
          'appInstalled',
          'standalone',
          'queryString',
        ],
        pwaHead: [
          {
            tagName: 'link',
            rel: 'icon',
            href: 'img/pwa/icon-192.png',
          },
          {
            tagName: 'link',
            rel: 'manifest',
            href: 'manifest.json',
          },
          {
            tagName: 'meta',
            name: 'theme-color',
            content: '#0066cc',
          },
          {
            tagName: 'meta',
            name: 'apple-mobile-web-app-capable',
            content: 'yes',
          },
        ],
      },
    ],
    [
      '@docusaurus/plugin-google-gtag',
      {
        trackingID: 'G-XXXXXXXXXX',
        anonymizeIP: true,
      },
    ],
    [
      '@docusaurus/plugin-vercel-analytics',
      {
        debug: false,
        mode: 'auto',
      },
    ],

    [
      '@docusaurus/plugin-dead-link-check',
      {
        checkInternal: true,
        checkExternal: true,
        exclude: [
          '/docs/K-ITIL-10_ITIL_MATRIX/**',
          '**/node_modules/**',
          '**/_*.{md,mdx}',
        ],
        timeout: 10000,
      },
    ],
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'versions',
        path: 'versions',
        routeBasePath: 'versions',
        sidebarPath: require.resolve('./sidebarsVersions.js'),
        versions: {
          current: {
            label: 'v1.0.0 (current)',
            path: '1.0.0',
          },
        },
      },
    ],
    '@docusaurus/plugin-content-pages',
    '@docusaurus/theme-mermaid',
    [
      'docusaurus-plugin-pdf',
      {
        exclude: [
          '**/_*.{md,mdx}',
          '**/node_modules/**',
          '**/K-ITIL-10_ITIL_MATRIX/**',
          '**/K-DEV-08_DEVELOPMENT/**',
          '**/README.md',
        ],
        include: [
          '**/K-CORE-01_INFRASTRUCTURE/**',
          '**/K-XRO-02_SUPER_AGENT/**',
          '**/K-KAI-03_ORCHESTRATION/**',
          '**/K-SOC-04_SECURITY/**',
          '**/K-NOC-05_OPERATIONS/**',
          '**/K-PSA-06_BUSINESS/**',
          '**/K-GRC-07_COMPLIANCE/**',
          '**/K-API-09_API_REFERENCE/**',
        ],
        pdfOptions: {
          format: 'A4',
          printBackground: true,
          margin: {
            top: '1in',
            bottom: '1in',
            left: '1in',
            right: '1in',
          },
        },
      },
    ],

    '@docusaurus/theme-live-codeblock',
    '@docusaurus/plugin-ideal-image',
    [
      'docusaurus-plugin-openapi-docs',
      {
        id: 'apiDocs',
        docsPluginId: 'classic',
        config: {
          provisioningApi: {
            specPath: 'docs/K-API-09_API_REFERENCE/K-API-OPEN-001_provisioning.yaml',
            outputDir: 'docs/K-API-09_API_REFERENCE/rest_api/provisioning',
            sidebarOptions: {
              groupPathsBy: 'tag',
              categoryLinkSource: 'tag',
            },
            downloadLink: 'https://raw.githubusercontent.com/kubric/platform/main/docs/K-API-09_API_REFERENCE/K-API-OPEN-001_provisioning.yaml',
            version: '1.0.0',
            label: 'Provisioning API',
            baseUrl: '/api/provisioning',
          },
          triageApi: {
            specPath: 'docs/K-API-09_API_REFERENCE/K-API-OPEN-002_triage.yaml',
            outputDir: 'docs/K-API-09_API_REFERENCE/rest_api/triage',
            sidebarOptions: {
              groupPathsBy: 'tag',
              categoryLinkSource: 'tag',
            },
            downloadLink: 'https://raw.githubusercontent.com/kubric/platform/main/docs/K-API-09_API_REFERENCE/K-API-OPEN-002_triage.yaml',
            version: '1.0.0',
            label: 'Triage API',
            baseUrl: '/api/triage',
          },
          billingApi: {
            specPath: 'docs/K-API-09_API_REFERENCE/K-API-OPEN-003_billing.yaml',
            outputDir: 'docs/K-API-09_API_REFERENCE/rest_api/billing',
            sidebarOptions: {
              groupPathsBy: 'tag',
              categoryLinkSource: 'tag',
            },
            downloadLink: 'https://raw.githubusercontent.com/kubric/platform/main/docs/K-API-09_API_REFERENCE/K-API-OPEN-003_billing.yaml',
            version: '1.0.0',
            label: 'Billing API',
            baseUrl: '/api/billing',
          },
        },
      },
    ],
    'docusaurus-theme-openapi-docs',

    '@docusaurus/theme-common',
    [
      '@docusaurus/remark-plugin-npm2yarn',
      {
        sync: true,
        converters: ['yarn', 'pnpm'],
      },
    ],
    'docusaurus-plugin-plantuml',
    '@docusaurus/plugin-tag',
    'docusaurus-plugin-sass',
    [
      'docusaurus-plugin-structured-data',
      {
        schemas: [
          {
            path: 'docs/K-CORE-01_INFRASTRUCTURE/K-HW-R740_HARDWARE/**/*.md',
            schema: 'HardwareSpec',
            outputPath: 'data/hardware-assets.json',
          },
          {
            path: 'docs/K-CORE-01_INFRASTRUCTURE/K-DL-POSTGRES/**/*.sql',
            schema: 'DatabaseSchema',
            outputPath: 'data/database-schemas.json',
          },
          {
            path: 'docs/K-NOC-05_OPERATIONS/K-NOC-CM-ANSIBLE/**/*.yml',
            schema: 'AnsiblePlaybook',
            outputPath: 'data/playbooks.json',
          },
        ],
      },
    ],

    [
      './plugins/kubric-sidebar-generator',
      {
        rootDir: 'docs',
        moduleOrder: [
          'K-CORE-01_INFRASTRUCTURE',
          'K-XRO-02_SUPER_AGENT',
          'K-KAI-03_ORCHESTRATION',
          'K-SOC-04_SECURITY',
          'K-NOC-05_OPERATIONS',
          'K-PSA-06_BUSINESS',
          'K-GRC-07_COMPLIANCE',
          'K-DEV-08_DEVELOPMENT',
          'K-API-09_API_REFERENCE',
          'K-ITIL-10_ITIL_MATRIX',
        ],
        enforceKPrefix: true,
        urlPrefix: 'docs',
      },
    ],
    [
      './plugins/kubric-itil-mapper',
      {
        matrixPath: 'docs/K-ITIL-10_ITIL_MATRIX',
        enforceLinks: true,
        codePaths: [
          'docs/K-CORE-01_INFRASTRUCTURE',
          'docs/K-XRO-02_SUPER_AGENT',
          'docs/K-KAI-03_ORCHESTRATION',
          'docs/K-SOC-04_SECURITY',
          'docs/K-NOC-05_OPERATIONS',
          'docs/K-PSA-06_BUSINESS',
          'docs/K-GRC-07_COMPLIANCE',
        ],
        generateEvidenceReport: true,
        evidenceOutputPath: 'data/compliance-evidence.json',
      },
    ],
    [
      './plugins/kubric-hardware-specs',
      {
        hardwarePath: 'docs/K-CORE-01_INFRASTRUCTURE/K-HW-R740_HARDWARE',
        renderTables: true,
        exportFormats: ['json', 'csv', 'markdown'],
        assetTracking: true,
        warrantyAlerts: true,
      },
    ],
    [
      './plugins/kubric-arch-diagrams',
      {
        mermaidTheme: 'dark',
        defaultZoom: 1.2,
        diagramPaths: [
          'docs/**/*.mmd',
          'docs/**/*.mermaid',
        ],
        exportFormat: 'svg',
      },
    ],
    [
      './plugins/kubric-schema-visualizer',
      {
        schemaPaths: [
          'docs/K-CORE-01_INFRASTRUCTURE/K-DL-POSTGRES/**/*.sql',
          'docs/K-CORE-01_INFRASTRUCTURE/K-DL-CLICKHOUSE/**/*.sql',
        ],
        generateERD: true,
        outputFormat: 'svg',
      },
    ],
    [
      './plugins/kubric-ansible-viewer',
      {
        playbookPaths: 'docs/K-NOC-05_OPERATIONS/K-NOC-CM-ANSIBLE',
        syntaxHighlight: true,
        executionDocs: true,
        variables: true,
      },
    ],
    [
      './plugins/kubric-telemetry-dashboard',
      {
        metricsPath: 'docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE',
        sampleData: true,
        visualizationType: 'grafana',
      },
    ],
    [
      './plugins/kubric-evidence-vault',
      {
        evidencePath: 'docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT',
        auditReady: true,
        exportFormats: ['pdf', 'json', 'csv'],
        chainOfCustody: true,
        blake3Hashes: true,
      },
    ],
    [
      './plugins/kubric-code-search',
      {
        codePaths: [
          'docs/K-XRO-02_SUPER_AGENT/**/*.rs',
          'docs/K-XRO-02_SUPER_AGENT/**/*.go',
          'docs/K-KAI-03_ORCHESTRATION/**/*.py',
          'docs/K-PSA-06_BUSINESS/**/*.tsx',
          'docs/K-PSA-06_BUSINESS/**/*.ts',
        ],
        indexComments: true,
        indexStrings: true,
      },
    ],
    'docusaurus-plugin-redoc',
    'docusaurus-plugin-matomo',
    'docusaurus-plugin-segment',
    'docusaurus-plugin-hotjar',
    [
      'docusaurus-plugin-google-adsense',
      {
        adClient: 'ca-pub-XXXXXXXXXXXXXXXX',
      },
    ],
    '@easyops-cn/docusaurus-search-local',
    'docusaurus-plugin-image-zoom',
    'docusaurus-plugin-sentry',
    'docusaurus-plugin-drawio',
    '@docusaurus/plugin-client-redirects',
    [
      'docusaurus-graphql-plugin',
      {
        schema: './graphql/schema.graphql',
        routeBasePath: '/graphql',
      },
    ],
    '@docusaurus/plugin-debug',
    [
      'docusaurus-plugin-umami',
      {
        scriptUrl: 'https://umami.example.com/umami.js',
        websiteId: 'UMAMI-ID',
      },
    ],
    [
      'docusaurus-plugin-fathom',
      {
        siteId: 'FATHOM-ID',
        spa: true,
      },
    ],
    [
      'docusaurus-plugin-plausible',
      {
        domain: 'docs.kubric.io',
      },
    ],
    [
      'docusaurus-plugin-yandex-metrica',
      {
        counterId: 'YANDEX-ID',
      },
    ],
  ],

  themes: [
    '@docusaurus/theme-mermaid',
    'docusaurus-theme-openapi-docs',
    '@docusaurus/theme-live-codeblock',
    '@docusaurus/theme-search-algolia',
    [
      '@docusaurus/theme-classic',
      {
        customCss: require.resolve('./src/css/custom.css'),
      },
    ],
  ],

  markdown: {
    mermaid: true,
    format: 'detect',
    mdx1Compat: false,
    remarkPlugins: [
      [require('@docusaurus/remark-plugin-npm2yarn'), { sync: true }],
    ],
    rehypePlugins: [],
  },

  stylesheets: [
    {
      href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap',
      type: 'text/css',
    },
    {
      href: 'https://cdn.jsdelivr.net/npm/katex@0.13.24/dist/katex.min.css',
      type: 'text/css',
      integrity: 'sha384-odtC+0UGzzFL/6PNoE8rX/SPcQDXBJ+uRepguP4QkPCm2LBxH3FA3y+fKSiJ+AmM',
      crossorigin: 'anonymous',
    },
  ],

  scripts: [
    {
      src: 'https://cdn.jsdelivr.net/npm/mermaid@10.6.1/dist/mermaid.min.js',
      async: true,
    },
  ],

  staticDirectories: ['static', 'public'],
  
  customFields: {
    kubricVersion: '1.0.0',
    kubricDocs: 'enterprise',
    supportedKernels: ['5.15+', '6.x'],
  },
};

export default config;

const fs = require('fs');
const path = require('path');

const pages = [
  { key: 'overview', relativePath: 'platform/index.mdx', title: 'Platform', description: 'Kubric platform overview.' },
  { key: 'soc', relativePath: 'platform/soc.mdx', title: 'Platform SOC', description: 'Kubric SOC platform overview.' },
  { key: 'noc', relativePath: 'platform/noc.mdx', title: 'Platform NOC', description: 'Kubric NOC platform overview.' },
  { key: 'grc', relativePath: 'platform/grc.mdx', title: 'Platform GRC', description: 'Kubric GRC platform overview.' },
  { key: 'psa', relativePath: 'platform/psa.mdx', title: 'Platform PSA', description: 'Kubric PSA platform overview.' },
  { key: 'kai', relativePath: 'platform/kai.mdx', title: 'Platform KAI', description: 'Kubric KAI platform overview.' },
];

const pagesRoot = path.join(__dirname, '..', 'src', 'pages');

for (const page of pages) {
  const targetFile = path.join(pagesRoot, page.relativePath);
  fs.mkdirSync(path.dirname(targetFile), { recursive: true });

  const output = `---\ntitle: ${page.title}\ndescription: ${page.description}\n---\n\nimport PlatformMarketingPage from '@site/src/components/marketing/PlatformMarketingPage';\nimport {platformPageByKey} from '@site/src/lib/marketing/platformContent';\n\n<PlatformMarketingPage page={platformPageByKey('${page.key}')} />\n`;

  fs.writeFileSync(targetFile, output, 'utf8');
  console.log(`Generated ${targetFile}`);
}

console.log('Platform marketing page generation complete.');

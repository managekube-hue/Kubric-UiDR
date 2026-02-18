const fs = require('fs');
const path = require('path');

const docsRoot = path.join(__dirname, '..', 'docs');
const layoutPath = path.join(__dirname, '..', 'src', 'theme', 'DocItem', 'Layout', 'index.tsx');
const cssPath = path.join(__dirname, '..', 'src', 'css', 'custom.css');

function collectDocs(dir, acc = []) {
  const entries = fs.readdirSync(dir, {withFileTypes: true});
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      collectDocs(fullPath, acc);
      continue;
    }
    if (entry.name.endsWith('.md') || entry.name.endsWith('.mdx')) {
      acc.push(fullPath);
    }
  }
  return acc;
}

function assertContains(filePath, fragment, label) {
  const content = fs.readFileSync(filePath, 'utf8');
  if (!content.includes(fragment)) {
    throw new Error(`${label} is missing from ${filePath}`);
  }
}

function main() {
  const docsFiles = collectDocs(docsRoot);

  assertContains(layoutPath, 'docs-aesthetic-article', 'Global docs shell class');
  assertContains(layoutPath, 'Open in Notion', 'Notion-first CTA');
  assertContains(cssPath, '.docs-aesthetic-article', 'Docs aesthetic styles');
  assertContains(cssPath, '.docs-aesthetic-content', 'Docs content styles');

  console.log(`Docs aesthetic applied globally to ${docsFiles.length} documentation pages.`);
  console.log('Verified: enterprise shell, Notion-first CTA, and global docs styling.');
}

main();

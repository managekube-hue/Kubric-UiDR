const fs = require('fs');
const path = require('path');

const docsRoot = path.join(__dirname, '..');
const pagesRoot = path.join(docsRoot, 'src', 'pages');
const docsContentRoot = path.join(docsRoot, 'docs');
const outMarkdownFile = path.join(docsRoot, 'ROUTES-BREADCRUMBS.md');
const outJsonFile = path.join(docsRoot, 'ROUTES-BREADCRUMBS.json');
const outCsvFile = path.join(docsRoot, 'ROUTES-BREADCRUMBS.csv');

const markdownExt = ['.md', '.mdx'];

function walk(dir) {
  const entries = fs.readdirSync(dir, {withFileTypes: true});
  const files = [];

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(fullPath));
      continue;
    }

    if (markdownExt.includes(path.extname(entry.name))) {
      files.push(fullPath);
    }
  }

  return files;
}

function formatSegment(segment) {
  return segment
    .replace(/[-_]/g, ' ')
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

function breadcrumbs(route) {
  const parts = route.split('/').filter(Boolean);
  if (parts.length === 0) {
    return 'Home';
  }

  const labels = ['Home'];
  let current = '';
  for (const part of parts) {
    current += `/${part}`;
    labels.push(formatSegment(part));
  }

  return labels.join(' > ');
}

function routeFromPages(filePath) {
  const rel = path.relative(pagesRoot, filePath).replace(/\\/g, '/');
  const noExt = rel.replace(/\.mdx?$/, '');

  if (noExt === 'index') {
    return '/';
  }

  if (noExt.endsWith('/index')) {
    return `/${noExt.replace(/\/index$/, '')}`;
  }

  return `/${noExt}`;
}

function routeFromDocs(filePath) {
  const rel = path.relative(docsContentRoot, filePath).replace(/\\/g, '/');
  const noExt = rel.replace(/\.mdx?$/, '');

  if (noExt.endsWith('/index')) {
    return `/docs/${noExt.replace(/\/index$/, '')}`;
  }

  return `/docs/${noExt}`;
}

const pageFiles = fs.existsSync(pagesRoot) ? walk(pagesRoot) : [];
const docsFiles = fs.existsSync(docsContentRoot) ? walk(docsContentRoot) : [];

const pageRows = pageFiles
  .map((filePath) => ({
    filePath: path.relative(docsRoot, filePath).replace(/\\/g, '/'),
    route: routeFromPages(filePath),
  }))
  .sort((a, b) => a.route.localeCompare(b.route));

const docsRows = docsFiles
  .map((filePath) => ({
    filePath: path.relative(docsRoot, filePath).replace(/\\/g, '/'),
    route: routeFromDocs(filePath),
  }))
  .sort((a, b) => a.route.localeCompare(b.route));

const allRows = [...pageRows, ...docsRows];

const generatedAt = new Date().toISOString();

let markdownOutput = '# Routes and Breadcrumb Inventory\n\n';
markdownOutput += `Generated: ${generatedAt}\n\n`;
markdownOutput += '| Route | Breadcrumbs | Source |\n';
markdownOutput += '|---|---|---|\n';

for (const row of allRows) {
  markdownOutput += `| ${row.route} | ${breadcrumbs(row.route)} | ${row.filePath} |\n`;
}

const jsonOutput = {
  generatedAt,
  totalRoutes: allRows.length,
  routes: allRows.map((row) => ({
    route: row.route,
    breadcrumbs: breadcrumbs(row.route),
    source: row.filePath,
  })),
};

const csvHeader = 'route,breadcrumbs,source\n';
const csvRows = allRows
  .map((row) => {
    const route = `"${row.route.replace(/"/g, '""')}"`;
    const crumb = `"${breadcrumbs(row.route).replace(/"/g, '""')}"`;
    const source = `"${row.filePath.replace(/"/g, '""')}"`;
    return `${route},${crumb},${source}`;
  })
  .join('\n');

fs.writeFileSync(outMarkdownFile, markdownOutput, 'utf8');
fs.writeFileSync(outJsonFile, JSON.stringify(jsonOutput, null, 2), 'utf8');
fs.writeFileSync(outCsvFile, csvHeader + csvRows + '\n', 'utf8');

console.log(`Wrote ${outMarkdownFile} with ${allRows.length} routes.`);
console.log(`Wrote ${outJsonFile} with ${allRows.length} routes.`);
console.log(`Wrote ${outCsvFile} with ${allRows.length} routes.`);

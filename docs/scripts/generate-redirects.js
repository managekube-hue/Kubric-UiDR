const fs = require('fs');
const path = require('path');
const manifest = require('../data/notion-pages-manifest.json');

const outRoot = path.join(__dirname, '..', 'src', 'pages');

manifest.forEach((entry) => {
  const target = entry.docPath; // e.g. /docs/.../Name
  // Build a file path under src/pages matching the route (strip leading slash)
  const filePath = path.join(outRoot, target.replace(/^\//, '')) + '.js';
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });

  const component = `import React, {useEffect} from 'react';\nexport default function Redirect() {\n  useEffect(() => {\n    // client-side redirect to docs route\n    window.location.replace('${target}');\n  }, []);\n  return (\n    <main style={{padding: '2rem'}}>\n      <h1>Redirecting...</h1>\n      <p>If you are not redirected, <a href="${target}">click here</a>.</p>\n    </main>\n  );\n}\n`;

  fs.writeFileSync(filePath, component, 'utf8');
  console.log('Wrote', filePath);
});

console.log('Redirect generation complete.');

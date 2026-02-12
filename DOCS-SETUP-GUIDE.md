# Documentation Setup Guide

## ✅ What's Been Done

1. **Docusaurus merged** into `/docs/` directory
2. **docu-notion GitHub Action** set up at `.github/workflows/docu-notion.yml`
3. **Documentation structure** ready for Vercel deployment
4. **Node.js dependencies** ready (yarn)

## 🔧 Next Steps (Required for Full Setup)

### Step 1: Set GitHub Secrets

The docu-notion GitHub Action requires secrets. Go to:

**Repository → Settings → Secrets and variables → Actions**

Add these secrets:

1. **`DOCU_NOTION_INTEGRATION_TOKEN`**
   - Get from: [Notion Developer Dashboard](https://www.notion.so/my-integrations)
   - Create integration, copy the token
   - Example: `secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

2. **`DOCU_NOTION_SAMPLE_ROOT_PAGE`**
   - The Notion page ID that contains your documentation
   - Example: `a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6`
   - From your Notion page URL: `https://notion.so/{PAGE_ID}?v=xxx`

3. **`VERCEL_TOKEN`**
   - Get from: [Vercel Account Settings](https://vercel.com/account/tokens)
   - Create new token (Authentication & Security)

4. **`VERCEL_ORG_ID`**
   - From: [Vercel Dashboard](https://vercel.com/dashboard) > Settings > Account

5. **`VERCEL_PROJECT_ID`**
   - After creating Vercel project (next step)

### Step 2: Import Kubric-UiDR to Vercel

1. Go to [Vercel Dashboard](https://vercel.com/dashboard)
2. Click **Add New** → **Project**
3. Import from GitHub repository `managekube-hue/Kubric-UiDR`
4. **Framework:** Select **Docusaurus**
5. **Root Directory:** Set to `docs/`
6. **Build Command:** `yarn build`
7. **Output Directory:** `.docusaurus` (Docusaurus default)
8. Click **Deploy**
9. Once created, get the **Project ID** from project settings and add to GitHub secrets as `VERCEL_PROJECT_ID`

### Step 3: Update Docusaurus Config (Optional)

The current `docs/docusaurus.config.js` has placeholder values. Update:

```javascript
const config = {
  title: 'Kubric Platform Documentation',
  tagline: 'Enterprise infrastructure automation & security orchestration',
  url: 'https://your-vercel-domain.vercel.app',
  baseUrl: '/',
  organizationName: 'managekube-hue',
  projectName: 'Kubric-UiDR',
  // ... rest of config
};
```

### Step 4: Test Locally (Optional)

Before pushing to GitHub:

```bash
cd docs
yarn install
yarn start
```

This opens documentation locally at `http://localhost:3000`

### Step 5: Commit and Push

```bash
cd /workspaces/Kubric-UiDR
git add .
git commit -m "Merge Docusaurus documentation and docu-notion GitHub Action

- Merge Kubric-docusaurs into /docs/ directory
- Add docu-notion GitHub Action for Notion→Docusaurus sync
- Configure Vercel deployment pipeline
- Set up yarn-based build system"
git push origin main
```

## 🔄 How It Works (After Setup)

### Automatic Sync Flow

1. **Edit in Notion** → Make changes to your documentation in Notion
2. **GitHub Action Triggers** → docu-notion.yml workflow runs automatically
3. **Pull from Notion** → `yarn pull` syncs content from Notion
4. **Auto-commit** → Changes automatically committed to GitHub
5. **Build Docs** → Docusaurus builds the site
6. **Deploy to Vercel** → Live documentation updated instantly

### Manual Trigger (Optional)

If you need to force a sync:
- Go to Actions tab in GitHub → docu-notion workflow → **Run workflow** → **Run workflow**

## 📁 Directory Structure

```
Kubric-UiDR/
├── docs/                          # ← Docusaurus site
│   ├── docusaurus.config.js       # Main Docusaurus config
│   ├── package.json               # Dependencies + "pull" script
│   ├── docs/                      # Synced content from Notion
│   ├── blog/                      # Blog posts
│   ├── src/                       # Custom React components
│   └── static/                    # Images, fonts, etc.
├── .github/
│   └── workflows/
│       └── docu-notion.yml        # ← GitHub Action for sync + deploy
├── K-CORE-01_INFRASTRUCTURE/      # Code modules...
├── K-XRO-02_SUPER_AGENT/
└── ... (other modules)
```

## 🐛 Troubleshooting

**GitHub Action fails with "yarn pull" error:**
- Check `DOCU_NOTION_INTEGRATION_TOKEN` is correct
- Verify `DOCU_NOTION_SAMPLE_ROOT_PAGE` points to correct Notion page
- Ensure Notion integration has access to that page

**Vercel deployment fails:**
- Check `VERCEL_TOKEN`, `VERCEL_ORG_ID`, `VERCEL_PROJECT_ID` are correct
- Verify Root Directory is set to `docs/` in Vercel project settings
- Check Build Command includes `yarn` installation

**Docusaurus build fails locally:**
- Delete `docs/node_modules` and `docs/yarn.lock`
- Run `cd docs && yarn install` again
- Check Node version: `node --version` (should be v14+)

## 📚 Resources

- [Docusaurus Docs](https://docusaurus.io/docs)
- [docu-notion GitHub](https://github.com/sillsdev/docu-notion)
- [Notion Integration Setup](https://developers.notion.com/docs/create-a-notion-integration)
- [Vercel Documentation](https://vercel.com/docs)

## ✨ What's Next

Once setup is complete, your team can:
1. Write/edit documentation directly in Notion
2. Changes automatically sync to GitHub and deploy to Vercel
3. Everything stays in sync across tools
4. No manual documentation maintenance needed

---

**Questions?** Check `.github/workflows/docu-notion.yml` for the exact automation sequence.

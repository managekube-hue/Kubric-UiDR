# AWS Sandbox Deploy (1-Hour Path)

## Recommended path
Use **AWS Amplify Hosting (SSR)** for fastest sandbox deployment of the Next.js interface.

## Do you need an AWS package/extension?
- Required for CLI-driven deploy: AWS CLI v2.
- Optional in VS Code: AWS Toolkit extension (helpful, not required).
- For fastest path right now: you can deploy from AWS Console without installing extension.

## Preconditions
1. Frontend build is passing locally (`npm run build` in `frontend`).
2. Repo is pushed to GitHub.
3. AWS account with permission to create Amplify app and IAM roles.

## Console steps (fast)
1. AWS Console -> Amplify -> New app -> Host web app.
2. Connect GitHub repo `managekube-hue/Kubric-UiDR`.
3. Branch: `main`.
4. Confirm Amplify picks monorepo config from `amplify.yml`.
5. Set environment variables:
   - `NEXT_PUBLIC_API_BASE`
   - `NEXT_PUBLIC_KAI_URL`
   - `NEXT_PUBLIC_NATS_WS_URL`
   - `AUTHENTIK_ISSUER` (if SSO enabled)
   - `AUTHENTIK_CLIENT_ID` (if SSO enabled)
   - `AUTHENTIK_CLIENT_SECRET` (if SSO enabled)
   - `NEXTAUTH_SECRET` (if SSO enabled)
   - `NEXTAUTH_URL` (Amplify URL)
6. Deploy.

## Sandbox URL test checklist
1. `/login` loads.
2. `/dashboard` renders.
3. `/agents` renders.
4. `/billing` renders.
5. `/detection` renders.
6. API auth route responds: `/api/auth/[...nextauth]`.

## If you prefer container deploy
Use `Dockerfile.web` with AWS App Runner or ECS Fargate.

Required extras for that path:
- Docker installed locally or CI build.
- ECR repo.
- App Runner/ECS service config.

Amplify is still the fastest path for the 1-hour target.

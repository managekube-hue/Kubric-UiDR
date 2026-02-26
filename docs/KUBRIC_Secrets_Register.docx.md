

| KUBRIC UIDR Secrets Register  \+  ERPNext Architecture Decision *Every API key, token, credential, and certificate — organised by Vault path  •  ERPNext vs Next.js customer portal decision* Store ALL of these in HashiCorp Vault.  Zero plaintext in env vars, config files, or code. |
| ----- |

# **The Golden Rule**

Every single credential in this document lives in HashiCorp Vault under the path shown. Your Go services fetch them via vault/api at startup. Your Python agents fetch them via hvac. Nothing goes in a .env file, K8s Secret manifest, docker-compose env block, or GitHub Actions secret except VAULT\_ADDR and VAULT\_TOKEN — and those two only in ephemeral CI contexts.

| Vault Path Convention secret/kubric/{category}/{service}  —  Example: secret/kubric/ti/otx  for AlienVault OTX key.All Kubric apps use KV v2. Mount point is 'secret'. Fetch with: client.KVv2("secret").Get(ctx, "kubric/ti/otx") |
| :---- |

# **1\. Core Platform Secrets**

These secrets are required on Day 1 before any service will start. Without them Layer 0 and Layer 1 cannot boot.

| 1.1  Database & Storage |
| :---- |

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Supabase DB Password** | secret/kubric/db/supabase | K-SVC, VDR, KIC, NOC, KAI | Supabase dashboard → Settings → Database → Database password | **YES** |
| **Supabase Anon Key** | secret/kubric/db/supabase\_anon | Next.js frontend | Supabase dashboard → Settings → API → anon public key | **YES** |
| **Supabase Service Role Key** | secret/kubric/db/supabase\_service | Backend services (admin DB access) | Supabase dashboard → Settings → API → service\_role key | **YES** |
| **ClickHouse Password** | secret/kubric/db/clickhouse | All services writing events | Set at ClickHouse install time or docker-compose env | **YES** |
| **Neo4j Password** | secret/kubric/db/neo4j | KAI identity graph (ITDR) | Set at Neo4j install. Default changed at first boot. | **YES** |
| **Redis Password** | secret/kubric/db/redis | Celery, session cache, TI dedup | Set at Redis install (requirepass in redis.conf) | **YES** |
| **MinIO Access Key** | secret/kubric/storage/minio\_access | NOC backup, PCAP, QBR PDFs | MinIO console → Identity → Service Accounts | **YES** |
| **MinIO Secret Key** | secret/kubric/storage/minio\_secret | NOC backup, PCAP, QBR PDFs | Same as above — generated at same time | **YES** |

| 1.2  Vault & Infrastructure |
| :---- |

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Vault Root Token** | NOT stored in Vault — kept offline | Vault init only | vault operator init → save unseal keys \+ root token offline in secure storage | **YES** |
| **Vault Unseal Keys (3 of 5\)** | NOT stored in Vault — split across 3 people | Vault restart/DR | vault operator init \-key-shares=5 \-key-threshold=3 | **YES** |
| **Vault K8s Auth Role** | secret/kubric/vault/k8s\_role | K8s pod auth to Vault | vault write auth/kubernetes/role/kubric ... (set up at K8s install) | **YES** |
| **Temporal DB Password** | secret/kubric/temporal/db | Temporal workflow engine | Set at Temporal PostgreSQL init | **YES** |
| **TUF Signing Key (ed25519)** | secret/kubric/tuf/signing\_key | Agent OTA update signing | Generate: cosign generate-key-pair → store private key in Vault | **YES** |
| **Docker Registry Token** | secret/kubric/registry/token | CI/CD image push | Gitea Settings → Applications → Access Tokens | **YES** |

# **2\. AI & LLM Secrets**

CrewAI itself has no API key — it is a local Python framework. These are the keys for the LLM backends that CrewAI agents call.

| 2.1  Local Inference (No Key Needed) |
| :---- |

| Ollama \+ vLLM — Zero Auth by Default Ollama running at http://localhost:11434 and vLLM at http://localhost:8000 require NO API key in dev. In production, put both behind Caddy with JWT auth and store the token at secret/kubric/llm/local\_token. Do NOT expose port 11434 or 8000 to the internet. |
| :---- |

| 2.2  Cloud LLM Fallbacks (Expensive — Use Sparingly) |
| :---- |

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **OpenAI API Key** | secret/kubric/llm/openai | KAI-TRIAGE fallback (GPT-4o) | platform.openai.com → API Keys → Create new secret key | OPTIONAL |
| **Anthropic API Key** | secret/kubric/llm/anthropic | KAI-KEEPER long-context analysis | console.anthropic.com → API Keys | OPTIONAL |
| **Cohere API Key** | secret/kubric/llm/cohere | RAG embeddings (multilingual) | dashboard.cohere.com → API Keys | OPTIONAL |
| **HuggingFace Token** | secret/kubric/llm/huggingface | Model downloads (gated models) | huggingface.co → Settings → Access Tokens → New token (read scope) | OPTIONAL |

| Cost Control Set hard spend limits on OpenAI and Anthropic dashboards. KAI should default to local Ollama. Cloud fallbacks only fire when Ollama/vLLM is unavailable or context window exceeds local model. Log every cloud call to ClickHouse audit table with token count. |
| :---- |

| 2.3  CrewAI & Composio |
| :---- |

CrewAI (the framework) has no API key. However two things it connects to do:

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Composio API Key** | secret/kubric/ai/composio | KAI agents → GitHub, Jira, Slack tools | app.composio.dev → Settings → API Keys | OPTIONAL |
| **Composio Connected Account OAuth tokens** | secret/kubric/ai/composio\_oauth/{tool} | Per-tool auth (GitHub, Slack etc) | Composio dashboard → Connected Accounts → each integration | OPTIONAL |
| **LangSmith API Key (tracing)** | secret/kubric/ai/langsmith | LangChain chain observability | smith.langchain.com → Settings → API Keys | OPTIONAL |
| **MLflow Tracking Password** | secret/kubric/ai/mlflow | ML experiment auth (if MLflow server secured) | Set at MLflow server install | OPTIONAL |

# **3\. Threat Intelligence API Keys**

These feed the TI pipeline in kai/intel/ti\_feeds.py. Without at least OTX and AbuseIPDB, the TI layer runs on CISA KEV \+ IPSum only (free, no key). For a production MSP platform you want all of them.

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **AlienVault OTX API Key** | secret/kubric/ti/otx | KAI TI feeds (15min poll) | otx.alienvault.com → Settings → API Key (free account) | **YES** |
| **AbuseIPDB API Key** | secret/kubric/ti/abuseipdb | KAI TI feeds (30min poll) | abuseipdb.com → User Account → API → Create Key (free tier: 1000/day) | **YES** |
| **MISP Auth Key** | secret/kubric/ti/misp | PyMISP client (1hr poll) | MISP instance → Administration → Users → Auth Keys | YES if MISP deployed |
| **MISP URL** | secret/kubric/ti/misp\_url | PyMISP client | Your MISP instance URL e.g. https://misp.yourdomain.com | YES if MISP deployed |
| **PhishTank App Key** | secret/kubric/ti/phishtank | SIDR phishing URL check | phishtank.org → Register → App key (free) | OPTIONAL |
| **HaveIBeenPwned API Key** | secret/kubric/ti/hibp | ITDR credential breach check | haveibeenpwned.com/API/Key — £3.50/month | OPTIONAL |
| **Shodan API Key** | secret/kubric/ti/shodan | GRC external attack surface | account.shodan.io → API Key (free tier limited) | OPTIONAL |
| **Censys API ID \+ Secret** | secret/kubric/ti/censys\_id \+ censys\_secret | GRC ASM module | censys.io → Account → API → App credentials | OPTIONAL |
| **GreyNoise API Key** | secret/kubric/ti/greynoise | Alert false-positive reduction | viz.greynoise.io → Account → API Key | OPTIONAL |
| **Wiz API Client ID** | secret/kubric/ti/wiz\_client\_id | CDR cloud threat intel | Wiz portal → Settings → Service Accounts | OPTIONAL |
| **Wiz API Secret** | secret/kubric/ti/wiz\_secret | CDR cloud threat intel | Same as above | OPTIONAL |
| **MaxMind License Key** | secret/kubric/ti/maxmind | GeoIP2 DB download (monthly) | maxmind.com → My Account → License Keys (free GeoLite2 account) | **YES** |

# **4\. Communication & Voice Secrets**

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Twilio Account SID** | secret/kubric/comms/twilio\_sid | KAI-COMM SMS/voice fallback | console.twilio.com → Account Info panel | **YES** |
| **Twilio Auth Token** | secret/kubric/comms/twilio\_token | KAI-COMM SMS/voice fallback | console.twilio.com → Account Info panel | **YES** |
| **Twilio Phone Number** | secret/kubric/comms/twilio\_number | Outbound SMS caller ID | Twilio → Phone Numbers → Active Numbers | **YES** |
| **Vapi API Key** | secret/kubric/comms/vapi | KAI-COMM AI voice agents | dashboard.vapi.ai → Organization → API Keys | OPTIONAL |
| **Vapi Phone Number ID** | secret/kubric/comms/vapi\_phone\_id | Vapi outbound calls | Vapi dashboard → Phone Numbers | OPTIONAL |
| **Slack Bot Token** | secret/kubric/comms/slack\_bot | KAI-COMM alert notifications | api.slack.com → Apps → OAuth & Permissions → Bot Token (xoxb-...) | OPTIONAL |
| **Slack Signing Secret** | secret/kubric/comms/slack\_signing | Slack webhook verification | api.slack.com → Apps → Basic Information → Signing Secret | OPTIONAL |
| **PagerDuty Integration Key** | secret/kubric/comms/pagerduty | NOC on-call alerting | PagerDuty → Services → Integrations → Events API v2 key | OPTIONAL |
| **SendGrid API Key** | secret/kubric/comms/sendgrid | KAI-COMM email alerts | app.sendgrid.com → Settings → API Keys | OPTIONAL |

# **5\. Billing & Payments**

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Stripe Secret Key** | secret/kubric/billing/stripe\_secret | K-SVC billing engine | dashboard.stripe.com → Developers → API Keys → Secret key (sk\_live\_... or sk\_test\_...) | **YES** |
| **Stripe Publishable Key** | secret/kubric/billing/stripe\_public | Next.js Stripe.js frontend | Same page — Publishable key (pk\_live\_...) | **YES** |
| **Stripe Webhook Secret** | secret/kubric/billing/stripe\_webhook | K-SVC /v1/billing/webhook verification | Stripe → Developers → Webhooks → Select endpoint → Signing secret (whsec\_...) | **YES** |
| **Stripe Restricted Key (metering)** | secret/kubric/billing/stripe\_metering | Usage record writes only | Stripe → API Keys → Create restricted key → Usage Records write permission | **YES** |

# **6\. PSA & ITSM Integration Secrets**

You need at least one PSA connected. Start with Zammad (free, self-hosted). Add ConnectWise / Autotask / HaloPSA when onboarding MSP customers who already use them.

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Zammad URL** | secret/kubric/psa/zammad\_url | KAI-COMM ticket creation | Your Zammad instance URL. Self-hosted: https://zammad.yourdomain.com | **YES** |
| **Zammad API Token** | secret/kubric/psa/zammad\_token | KAI-COMM ticket creation | Zammad → Avatar → Profile → Token Access → Create Token | **YES** |
| **ConnectWise Company ID** | secret/kubric/psa/cw\_company | KAI-COMM → ConnectWise | Your CW company name (same as login company ID) | OPTIONAL |
| **ConnectWise Public Key** | secret/kubric/psa/cw\_public | KAI-COMM → ConnectWise | CW Manage → System → Members → API Members → Create member | OPTIONAL |
| **ConnectWise Private Key** | secret/kubric/psa/cw\_private | KAI-COMM → ConnectWise | Same as above | OPTIONAL |
| **Autotask API Username** | secret/kubric/psa/autotask\_user | KAI-COMM → Autotask | Autotask → Admin → Resources → API User | OPTIONAL |
| **Autotask API Secret** | secret/kubric/psa/autotask\_secret | KAI-COMM → Autotask | Same as above — Integration key | OPTIONAL |
| **HaloPSA Client ID** | secret/kubric/psa/halo\_client\_id | KAI-COMM → HaloPSA | HaloPSA → Configuration → Integrations → API | OPTIONAL |
| **HaloPSA Client Secret** | secret/kubric/psa/halo\_secret | KAI-COMM → HaloPSA | Same as above | OPTIONAL |

# **7\. Cloud Provider Credentials (CDR Module)**

CloudQuery needs read-only access to pull cloud asset inventory. Never use root/admin credentials — create dedicated read-only service accounts/roles.

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **AWS Access Key ID** | secret/kubric/cloud/aws\_access\_key | CDR CloudQuery, n8n AWS polling | AWS IAM → Users → Create user → SecurityAudit policy (read-only) | OPTIONAL |
| **AWS Secret Access Key** | secret/kubric/cloud/aws\_secret | CDR CloudQuery, n8n AWS polling | Same IAM user — download CSV | OPTIONAL |
| **Azure Client ID** | secret/kubric/cloud/azure\_client\_id | CDR CloudQuery, BloodHound Azure | Azure → Entra ID → App Registrations → New registration → Reader role | OPTIONAL |
| **Azure Client Secret** | secret/kubric/cloud/azure\_secret | CDR CloudQuery | Same app registration → Certificates & Secrets | OPTIONAL |
| **Azure Tenant ID** | secret/kubric/cloud/azure\_tenant | CDR CloudQuery | Azure → Entra ID → Overview → Tenant ID | OPTIONAL |
| **GCP Service Account JSON** | secret/kubric/cloud/gcp\_sa\_json | CDR CloudQuery, Google Workspace | GCP → IAM → Service Accounts → Create → Viewer role → Key JSON | OPTIONAL |
| **Google Workspace Admin Email** | secret/kubric/cloud/gws\_admin | SDR n8n Google Workspace polling | The delegated admin email for domain-wide delegation | OPTIONAL |

# **8\. Microsoft 365 / Entra ID (SDR Module)**

These enable the n8n O365 polling workflow and Wazuh O365 module subprocess. Requires an Entra ID app registration with specific Graph API permissions.

| Required Graph API Permissions (Application, not Delegated) AuditLog.Read.All  |  SecurityEvents.Read.All  |  ThreatIndicators.Read.All  |  MailboxSettings.Read  |  User.Read.All  |  Directory.Read.All  — All read-only. Admin consent required. |
| :---- |

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Entra App Client ID** | secret/kubric/o365/client\_id | n8n O365 workflow, Wazuh subprocess | Azure → Entra ID → App Registrations → New → copy Application (client) ID | YES for SDR |
| **Entra App Client Secret** | secret/kubric/o365/client\_secret | n8n O365 workflow, Wazuh subprocess | Same app → Certificates & Secrets → New client secret | YES for SDR |
| **Entra Tenant ID** | secret/kubric/o365/tenant\_id | n8n O365 workflow | Azure → Entra ID → Overview → Directory (tenant) ID | YES for SDR |
| **Exchange Online OAuth Token** | secret/kubric/o365/exchange\_token | KAI-COMM email analysis | Refreshed automatically from above client credentials | Derived |

# **9\. Identity & Auth Secrets**

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Authentik Bootstrap Token** | secret/kubric/auth/authentik\_bootstrap | Authentik first-run setup | Generated at Authentik first boot — copy from logs immediately | **YES** |
| **Authentik Secret Key** | secret/kubric/auth/authentik\_secret | Authentik signing | Random 50-char string — generated at install | **YES** |
| **JWT Signing Secret** | secret/kubric/auth/jwt\_secret | All Go services (jwtauth middleware) | Generate: openssl rand \-hex 32 | **YES** |
| **JWT Public Key (RS256)** | secret/kubric/auth/jwt\_public | Token verification | If using RS256 asymmetric — store public key here | OPTIONAL |
| **Casdoor Client ID** | secret/kubric/auth/casdoor\_client | Lightweight portal auth alternative | Casdoor → Applications → Create → Client ID | OPTIONAL |
| **Casdoor Client Secret** | secret/kubric/auth/casdoor\_secret | Lightweight portal auth alternative | Same application page | OPTIONAL |
| **Vault AppRole Role ID** | secret/kubric/vault/approle\_role\_id | Non-K8s service auth to Vault | vault read auth/approle/role/kubric/role-id | **YES** |
| **Vault AppRole Secret ID** | secret/kubric/vault/approle\_secret\_id | Non-K8s service auth to Vault | vault write \-f auth/approle/role/kubric/secret-id | **YES** |

# **10\. CI/CD & DevOps Secrets**

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Gitea Admin Token** | secret/kubric/cicd/gitea\_admin | Woodpecker CI → Gitea OAuth | Gitea → Settings → Applications → Generate Token | **YES** |
| **Woodpecker Agent Secret** | secret/kubric/cicd/woodpecker\_agent | Woodpecker agent ↔ server | woodpecker-cli setup → agent secret | **YES** |
| **Snyk Token** | secret/kubric/cicd/snyk | Dep vuln scanning in CI | app.snyk.io → Account Settings → Auth Token | OPTIONAL |
| **SonarQube Token** | secret/kubric/cicd/sonarqube | Static code analysis in CI | SonarQube → My Account → Security → Generate Tokens | OPTIONAL |
| **Cosign Key Pair** | secret/kubric/cicd/cosign\_private | Container image signing | cosign generate-key-pair → store cosign.key in Vault | **YES** |
| **ArgoCD Admin Password** | secret/kubric/cicd/argocd\_admin | GitOps deployment management | Set at ArgoCD install: argocd admin initial-password | **YES** |
| **Harbor Registry Credentials** | secret/kubric/cicd/harbor\_user \+ harbor\_pass | Private container registry | Harbor → Users → Create (if using Harbor instead of Gitea packages) | OPTIONAL |

# **11\. n8n Workflow Automation Secrets**

n8n itself stores credentials encrypted in its own database. The Vault integration means n8n never sees the raw values — it gets them injected at startup via the n8n-vault-init.sh script. Here is the complete list of what n8n needs.

| n8n Does NOT Have Its Own API Key n8n is self-hosted. You access it with Basic Auth credentials (set in docker-compose). The API key concept in n8n is for authenticating \*calls to\* the n8n API — not for n8n itself to authenticate to Anthropic or OpenAI. |
| :---- |

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **n8n Basic Auth User** | secret/kubric/n8n/basic\_auth\_user | n8n web UI login | Set at docker-compose deploy time — pick a strong username | **YES** |
| **n8n Basic Auth Password** | secret/kubric/n8n/basic\_auth\_password | n8n web UI login | Set at deploy time | **YES** |
| **n8n Encryption Key** | secret/kubric/n8n/encryption\_key | n8n credential DB encryption | n8n auto-generates on first start — back it up immediately | **YES** |
| **n8n API Key (for programmatic use)** | secret/kubric/n8n/api\_key | KAI → trigger n8n workflows via REST | n8n → Settings → API → Create API Key | OPTIONAL |
| **O365 credentials (injected into n8n)** | secret/kubric/o365/\* | n8n O365 workflow | See Section 8 — injected via n8n-vault-init.sh | YES for SDR |
| **Google Workspace credentials** | secret/kubric/cloud/gcp\_sa\_json | n8n GWS workflow | See Section 7 — injected via n8n-vault-init.sh | OPTIONAL |
| **Webhook signing secrets** | secret/kubric/n8n/webhook\_secret | n8n inbound webhook auth | Generate: openssl rand \-hex 32 | **YES** |

# **12\. Observability & Monitoring Secrets**

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **Grafana Admin Password** | secret/kubric/obs/grafana\_admin | Grafana web UI | Set at Grafana install. Change default 'admin'. | **YES** |
| **Grafana Service Account Token** | secret/kubric/obs/grafana\_token | Dashboard provisioning API | Grafana → Administration → Service Accounts → Add token | **YES** |
| **Loki Basic Auth** | secret/kubric/obs/loki\_auth | Log query auth (if Loki secured) | Set at Loki install with basic\_auth\_enabled | OPTIONAL |
| **Alertmanager Slack Webhook** | secret/kubric/obs/alertmanager\_slack | Platform alerts → Slack | Slack → Apps → Incoming Webhooks → Add | OPTIONAL |
| **PagerDuty Routing Key** | secret/kubric/obs/pagerduty\_routing | Platform critical alerts → PagerDuty | PagerDuty → Services → Integrations → Events API v2 | OPTIONAL |
| **Thanos Object Store Key** | secret/kubric/obs/thanos\_s3 | Long-term Prometheus storage | MinIO access key/secret — same as MinIO creds or dedicated bucket user | YES for prod |

# **13\. Compliance & GRC Secrets**

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **OpenSCAP XCCDF Signing Key** | secret/kubric/compliance/xccdf\_key | KIC compliance evidence signing | Generate Ed25519 key pair: ssh-keygen \-t ed25519 | OPTIONAL |
| **RegScale API Token** | secret/kubric/compliance/regscale | KIC OSCAL → RegScale import | RegScale portal → API Key | OPTIONAL |
| **OSV API Key (if paid)** | secret/kubric/compliance/osv | Supply chain vuln check | osv.dev — free tier needs no key. Paid tier key from console. | NO |
| **Dependency Track API Key** | secret/kubric/compliance/deptrack | SBOM vulnerability tracking | Dependency-Track → Administration → Access Management → Teams → API Key | OPTIONAL |

# **14\. Secrets by Priority — What to Set Up First**

| Priority | Secrets (must have these before that layer works) |
| ----- | ----- |
| **Day 1Layer 0** | Supabase password \+ anon key \+ service role key  •  ClickHouse password  •  Vault root token \+ unseal keys  •  JWT signing secret  •  MinIO access+secret  •  Gitea admin token  •  Woodpecker agent secret |
| **Week 1Layer 1** | Stripe secret \+ publishable \+ webhook  •  Authentik bootstrap \+ secret  •  Zammad URL \+ token  •  Temporal DB password  •  Neo4j password  •  Redis password |
| **Week 2Layer 2** | Twilio SID \+ token \+ number  •  OTX API key  •  AbuseIPDB key  •  MaxMind license  •  Ollama (no key in dev)  •  n8n encryption key \+ basic auth  •  Vault AppRole for KAI Python |
| **Week 3Layer 3** | MISP URL \+ auth key  •  O365 Client ID \+ Secret \+ Tenant ID  •  PhishTank key  •  HIBP key  •  TUF signing key  •  Cosign key pair  •  Snyk token |
| **Layer 4+Cloud/PSA** | AWS/Azure/GCP keys (CDR)  •  ConnectWise/Autotask/HaloPSA (per customer PSA)  •  Vapi key  •  PagerDuty  •  Grafana service token  •  OpenAI/Anthropic/Cohere (cloud LLM fallbacks) |

# **15\. ERPNext — Architecture Decision**

Short answer: ERPNext is in the architecture document as DocType JSON schemas vendored into K-SVC for case management structure (MDR module). It is NOT used as the customer portal. Here is the full decision with both options laid out.

## **What the Document Already Uses ERPNext For**

| Current Use — Vendor DocType Schemas Only vendor/frappe/erpnext/doctypes/\*\*/\*.json → These JSON files define Frappe DocType schemas (Customer, Contract, Invoice, SupportTicket, Project) which are mapped to K-SVC Go structs. This means Kubric uses ERPNext's battle-tested data MODEL without running the ERPNext application. MIT license — copy freely. |
| :---- |

## **Should ERPNext Replace or Extend the Customer Portal?**

There are three distinct options. They are not mutually exclusive.

### **Option A — Current Plan: Next.js Portal \+ ERPNext Schemas as Data**

This is what the architecture document specifies. You build the customer portal in Next.js, you model your data structures using ERPNext's DocType schemas as a reference, and ERPNext itself never runs in production.

| What You Get | What You Sacrifice |
| ----- | ----- |
| Full control over UI/UX — your brand, your design No Python overhead — Next.js \+ Tremor is fast Supabase \+ ClickHouse as data layer — no Frappe DB KiSS scorecard, real-time NATS alerts, Tremor dashboards | You build all accounting/invoicing from scratch No built-in CRM, contract management, HR More frontend code to write and maintain |

### **Option B — Run ERPNext as Internal Back-Office (Separate from Customer Portal)**

Run a full ERPNext/Frappe instance internally for your own business: accounting, payroll, CRM, contracts, support tickets. The customer-facing portal stays in Next.js. Kubric's Go services talk to ERPNext via its REST API for billing reconciliation.

| What You Get | What You Sacrifice |
| ----- | ----- |
| Full ERP: accounting, payroll, CRM, contracts, HR, inventory KAI-CLERK can write invoices directly to ERPNext via frappe-client Python library Handles your OWN company finances — not customer-facing MIT license — free self-hosted, no per-seat cost | Another system to maintain and back up Python/MariaDB stack vs your Go/Postgres stack — operational divergence Frappe has a steep learning curve for customisation |

If you choose Option B, add these libraries:

| Library | Install | Use |
| ----- | ----- | ----- |
| **frappe-client (Python)** | pip install frappe-client | KAI-CLERK → create Sales Invoice, update Payment Entry in ERPNext |
| **frappeclient (Go)** | github.com/lalit-rao/frappeclient | K-SVC → sync contract data to ERPNext Customer doctype |
| **ERPNext DocType REST API** | HTTP client only | GET/POST /api/resource/Customer — no library needed for basic CRUD |

### **Option C — ERPNext as the Customer Portal (NOT Recommended)**

Frappe has a customer portal built into ERPNext. You could expose it to customers for self-service ticket viewing and invoice download.

| Why This Is NOT Recommended for Kubric The ERPNext customer portal cannot render KiSS scorecards, real-time NATS alert feeds, compliance gap dashboards, or Tremor analytics charts. Frappe is a Python/Jinja2 stack — it cannot connect to your NATS event bus or render your ClickHouse telemetry. You would have a portal that shows invoices but not the security product your customers are paying for. Next.js is the correct choice for the security product UI. |
| :---- |

## **Recommendation**

| Do both A and B. They serve completely different purposes. Option A (Next.js portal) — for your CUSTOMERS to see their security posture, KiSS scores, alerts, compliance gaps, and agent health. This is your product. It must be fast, beautiful, and real-time. Option B (ERPNext internal) — for YOUR COMPANY to manage accounting, payroll, customer contracts, CRM, support tickets, and billing reconciliation. This is your ERP. KAI-CLERK writes invoices to it automatically. Add to requirements.txt:  pip install frappe-client Add ERPNext secrets to Vault:  secret/kubric/erp/url  \+  secret/kubric/erp/api\_key  \+  secret/kubric/erp/api\_secret |
| :---- |

## **ERPNext Secrets (if Option B deployed)**

| Secret / Credential | Vault Path | Used By | How to Obtain | Required? |
| ----- | ----- | ----- | ----- | ----- |
| **ERPNext URL** | secret/kubric/erp/url | KAI-CLERK, K-SVC | Your ERPNext instance: https://erp.yourdomain.com | YES if deployed |
| **ERPNext API Key** | secret/kubric/erp/api\_key | KAI-CLERK, K-SVC | ERPNext → User → API Access → Generate Keys | YES if deployed |
| **ERPNext API Secret** | secret/kubric/erp/api\_secret | KAI-CLERK, K-SVC | Same page — shown once, store immediately in Vault | YES if deployed |
| **ERPNext Admin Password** | secret/kubric/erp/admin\_password | Initial setup only | Set at ERPNext install. frappe bench new-site. | YES if deployed |
| **MariaDB Root Password** | secret/kubric/erp/mariadb\_root | ERPNext DB (MariaDB) | Set at MariaDB install | YES if deployed |
| **MariaDB ERPNext User Password** | secret/kubric/erp/mariadb\_user | ERPNext DB access | Created during frappe bench setup | YES if deployed |

# **Quick Count**

| Category | Required (YES) | Optional |
| ----- | ----- | ----- |
| Core Platform (DB, Vault, Storage) | 8 | 0 |
| AI & LLM | 0 | 7 |
| Threat Intelligence | 3 | 9 |
| Communications & Voice | 3 | 6 |
| Billing (Stripe) | 4 | 0 |
| PSA / ITSM | 2 | 6 |
| Cloud Providers (CDR) | 0 | 7 |
| Microsoft 365 (SDR) | 3 | 1 |
| Identity & Auth | 6 | 2 |
| CI/CD & DevOps | 5 | 2 |
| n8n Automation | 4 | 3 |
| Observability | 3 | 3 |
| Compliance / GRC | 0 | 4 |
| ERPNext (if deployed) | 6 | 0 |
| **TOTAL** | **47** | **50+** |

| The 47 Required Secrets Are What You Need to Operate The 50+ optional secrets expand capability but nothing breaks without them. Start with the Day 1 list from Section 14, then add by layer as you build. Every single one goes into Vault before any code reads it. |
| :---- |

KUBRIC UIDR — Secrets Register v1.0  —  Confidential  —  Keep offline backup of all Vault unseal keys
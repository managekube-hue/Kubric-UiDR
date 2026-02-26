# K-SEC-005 — Vault Policies (reference stub)
#
# Production policies: config/vault/policies.hcl
#
# Apply:  vault policy write kubric-default config/vault/policies.hcl
#
# Policy summary:
#   secret/data/agents/*           read         Rust agents
#   secret/data/nats/*             read         All services
#   database/creds/kubric-*        read         Dynamic DB creds per service
#   pki_int/issue/kubric-*         create,update   mTLS cert issuance
#   transit/encrypt/blake3-audit   create,update   HSM-backed signing
#   transit/sign/kubric-agent-*    create,update   Agent binary signing
#   secret/data/stripe/*           read         Billing (K-SVC, KAI)
#   secret/data/ti/*               read         Threat intelligence
#   secret/data/llm/*              read         LLM API keys
#   secret/data/psa/*              read         Zammad, n8n
#   secret/data/comm/*             read         Vapi, Twilio
#   secret/data/minio/*            read         S3 object storage
#   secret/metadata/*              list,read    Secret metadata browsing
#   auth/token/create-orphan       deny         Prevent root token creation

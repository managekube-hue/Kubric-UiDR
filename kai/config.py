from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """
    KAI runtime configuration.
    All values are read from environment variables (uppercase).
    Vault injects these at container startup in production.
    """

    model_config = SettingsConfigDict(env_prefix="KUBRIC_", case_sensitive=False)

    # Tenant — must always be set; drives NATS subject routing
    tenant_id: str

    # NATS
    nats_url: str = "nats://127.0.0.1:4222"

    # Databases
    clickhouse_url: str = "clickhouse://default:@127.0.0.1:9000/kubric"
    database_url: str = "postgresql://kubric:kubric@127.0.0.1:5432/kubric"

    # Inference — local first, cloud as fallback
    ollama_url: str = "http://127.0.0.1:11434"
    vllm_url: str = "http://127.0.0.1:8000"
    openai_api_key: str = ""      # cloud fallback only
    anthropic_api_key: str = ""   # cloud fallback only

    # API server
    api_host: str = "0.0.0.0"
    api_port: int = 8100
    log_level: str = "INFO"

    # n8n workflow automation (ITSM/alert routing)
    n8n_base_url: str = "http://n8n:5678"      # KUBRIC_N8N_BASE_URL
    n8n_webhook_url: str = ""                  # KUBRIC_N8N_WEBHOOK_URL (override)

    # LLM model name for Ollama
    model_name: str = "llama3.2"               # KUBRIC_MODEL_NAME

    # Layer 1 service URLs (KAI calls their REST APIs for scoring)
    vdr_url: str = "http://127.0.0.1:8081"     # KUBRIC_VDR_URL
    kic_url: str = "http://127.0.0.1:8082"     # KUBRIC_KIC_URL
    noc_url: str = "http://127.0.0.1:8083"     # KUBRIC_NOC_URL
    ksvc_url: str = "http://127.0.0.1:8080"    # KUBRIC_KSVC_URL

    # Temporal durable workflows
    temporal_address: str = "127.0.0.1:7233"   # KUBRIC_TEMPORAL_ADDRESS

    # Vapi voice alerts (KAI-COMM)
    vapi_api_key: str = ""                     # KUBRIC_VAPI_API_KEY
    vapi_phone_number_id: str = ""             # KUBRIC_VAPI_PHONE_NUMBER_ID
    alert_phone_number: str = ""               # KUBRIC_ALERT_PHONE_NUMBER

    # Stripe metered billing (KAI-CLERK)
    stripe_api_key: str = ""                   # KUBRIC_STRIPE_API_KEY

    # Zammad ITSM (KAI-CLERK)
    zammad_url: str = ""                       # KUBRIC_ZAMMAD_URL
    zammad_token: str = ""                     # KUBRIC_ZAMMAD_TOKEN

    # Composio tool-bridge (KAI core — GitHub, Jira, Slack integrations)
    composio_api_key: str = ""                 # KUBRIC_COMPOSIO_API_KEY


# Module-level singleton — import this everywhere in kai/
settings = Settings()  # type: ignore[call-arg]

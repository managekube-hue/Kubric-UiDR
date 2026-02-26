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


# Module-level singleton — import this everywhere in kai/
settings = Settings()  # type: ignore[call-arg]

# Vault dev-mode config placeholder
# In dev mode (-dev flag), Vault ignores this directory.
# For production, place vault.hcl here with storage, listener, and seal stanzas.
#
# Example production config:
# storage "raft" {
#   path = "/vault/data"
#   node_id = "kubric-vault-0"
# }
# listener "tcp" {
#   address     = "0.0.0.0:8200"
#   tls_disable = "true"   # Set false in prod; add tls_cert_file / tls_key_file
# }
# api_addr     = "http://0.0.0.0:8200"
# cluster_addr = "https://0.0.0.0:8201"
# ui           = true

# PSA - Customer Portal

**Module:** K-PSA-06  
**Purpose:** Customer self-service portal

---

## Overview

The customer portal provides self-service access to:
- Issue tracking
- Security posture
- Compliance reports
- Billing and invoices
- Knowledge base
- Asset management

---

## Portal Access

### URL
```
https://portal.kubric.security
```

### Login
- **Method:** OAuth2 (GitHub, Google, SSO)
- **Tenant Isolation:** Automatic based on user email domain
- **Roles:** Admin, Analyst, Viewer

---

## Features

### 1. Dashboard
- Security posture score
- Open issues count
- Recent alerts
- Compliance status
- Agent health

### 2. Issues
- Create new issue
- Track existing issues
- View issue history
- Attach files
- Comment and collaborate

### 3. Security
- Real-time alerts
- Detection coverage
- Threat intelligence
- Incident timeline

### 4. Compliance
- Framework status (NIST, PCI, ISO, SOC 2)
- Evidence vault
- Audit reports
- Control mapping

### 5. Assets
- Deployed agents
- Agent health status
- Coverage map
- Configuration

### 6. Billing
- Current usage
- Invoice history
- Payment methods
- Usage trends

### 7. Knowledge Base
- Documentation
- Runbooks
- FAQs
- Video tutorials

---

## API Access

### Portal API
```
Base URL: https://portal.kubric.security/api
```

### Authentication
```bash
# Get access token
curl -X POST https://portal.kubric.security/api/auth/token \
  -H "Content-Type: application/json" \
  -d '{"email": "user@acme.com", "password": "***"}'

# Use token
curl https://portal.kubric.security/api/v1/dashboard \
  -H "Authorization: Bearer <token>"
```

### Endpoints
```
GET  /api/v1/dashboard          # Dashboard data
GET  /api/v1/issues             # List issues
POST /api/v1/issues             # Create issue
GET  /api/v1/security/alerts    # Recent alerts
GET  /api/v1/compliance/status  # Compliance posture
GET  /api/v1/assets             # Agent list
GET  /api/v1/billing/usage      # Current usage
```

---

## Webhooks

### Configure Webhooks
```bash
# Create webhook
curl -X POST https://portal.kubric.security/api/v1/webhooks \
  -H "Authorization: Bearer <token>" \
  -d '{
    "url": "https://your-app.com/webhook",
    "events": ["issue.created", "alert.critical"],
    "secret": "webhook_secret"
  }'
```

### Webhook Events
- `issue.created` - New issue created
- `issue.updated` - Issue status changed
- `alert.critical` - Critical security alert
- `compliance.drift` - Compliance drift detected
- `agent.offline` - Agent went offline
- `invoice.generated` - New invoice created

---

## Customization

### Branding
- Custom logo
- Custom colors
- Custom domain
- White-label option

### Notifications
- Email notifications
- Slack integration
- Teams integration
- SMS alerts (via Twilio)

---

## Mobile App

### Features
- View dashboard
- Track issues
- Receive push notifications
- View alerts
- Check agent status

### Download
- iOS: App Store
- Android: Google Play

---

## Related Documentation

- [ITSM Integration](ITSM.md)
- [API Reference](../api-reference/)
- [ERPNext Integration](../docs/ERPNEXT-INTEGRATION.md)

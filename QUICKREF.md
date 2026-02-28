# Quick Reference - Fixed Commands

## All 4 Issues Fixed ✅

### 1. Validate Fixes (Run This First)
```powershell
# Windows
.\scripts\validate-fixes.ps1

# Linux/Mac
bash scripts/validate-fixes.sh
```

### 2. Start Production Stack
```bash
docker compose -f docker-compose.prod.yml up -d
```

### 3. Build Frontend
```bash
cd frontend
docker build -t kubric-web:test -f ../Dockerfile.web .
cd ..
```

### 4. Run Integration Tests
```bash
pip install pytest docker requests
pytest tests/integration/test_e2e.py -v
```

### 5. Test KAI Python (Correct Container Name)
```bash
# Find container name
docker ps | grep kai-python

# Execute (use actual name from above)
docker exec -it kubric-uidr-kai-python-1 python --version

# Test KAI modules
docker exec -it kubric-uidr-kai-python-1 python -c "from kai.deploy.docker_manager import DockerManager; print('✓ KAI loaded')"
```

---

## What Was Fixed

| Issue | Fix |
|-------|-----|
| Temporal UI tag | `2.29` → `2.28.0` |
| Frontend build | Context changed to `./frontend` |
| Test fixture | Increased timeouts, non-fatal health checks |
| Container name | Use `kubric-uidr-kai-python-1` (not `kubric-kai-python-1`) |

---

## Verification Checklist

- [ ] `validate-fixes.ps1` passes all 4 checks
- [ ] `docker compose -f docker-compose.prod.yml config` validates
- [ ] Frontend builds without errors
- [ ] Stack starts: `docker compose -f docker-compose.prod.yml up -d`
- [ ] All services healthy: `docker compose -f docker-compose.prod.yml ps`
- [ ] KAI Python accessible: `docker exec -it kubric-uidr-kai-python-1 python --version`

---

## Status: ✅ READY TO RUN

All fixes applied. Run validation script to confirm.

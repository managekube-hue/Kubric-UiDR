"""
ERPNext ITSM Bridge for KAI

Provides customer portal integration, billing, and full ITSM features.
"""

import httpx
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime

logger = logging.getLogger(__name__)


class ERPNextClient:
    """ERPNext REST API client for ITSM and customer portal integration"""
    
    def __init__(
        self,
        base_url: str,
        api_key: str,
        api_secret: str,
        timeout: int = 30
    ):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.api_secret = api_secret
        self.timeout = timeout
        self.client = httpx.AsyncClient(
            base_url=self.base_url,
            headers={
                "Authorization": f"token {api_key}:{api_secret}",
                "Content-Type": "application/json"
            },
            timeout=timeout
        )
    
    async def create_issue(
        self,
        subject: str,
        description: str,
        customer: str,
        priority: str = "Medium",
        issue_type: str = "Bug",
        **kwargs
    ) -> Dict[str, Any]:
        """
        Create issue in ERPNext
        
        Args:
            subject: Issue title
            description: Issue details
            customer: Customer ID
            priority: Low/Medium/High/Critical
            issue_type: Bug/Feature/Question/Incident
            
        Returns:
            Created issue document
        """
        data = {
            "doctype": "Issue",
            "subject": subject,
            "description": description,
            "customer": customer,
            "priority": priority,
            "issue_type": issue_type,
            "status": "Open",
            "raised_by": kwargs.get("raised_by", ""),
            **kwargs
        }
        
        response = await self.client.post("/api/resource/Issue", json=data)
        response.raise_for_status()
        return response.json()["data"]
    
    async def get_issue(self, issue_id: str) -> Dict[str, Any]:
        """Get issue by ID"""
        response = await self.client.get(f"/api/resource/Issue/{issue_id}")
        response.raise_for_status()
        return response.json()["data"]
    
    async def update_issue(
        self,
        issue_id: str,
        **updates
    ) -> Dict[str, Any]:
        """Update issue fields"""
        response = await self.client.put(
            f"/api/resource/Issue/{issue_id}",
            json=updates
        )
        response.raise_for_status()
        return response.json()["data"]
    
    async def add_comment(
        self,
        issue_id: str,
        comment: str,
        comment_by: str
    ) -> Dict[str, Any]:
        """Add comment to issue"""
        data = {
            "doctype": "Comment",
            "comment_type": "Comment",
            "reference_doctype": "Issue",
            "reference_name": issue_id,
            "content": comment,
            "comment_by": comment_by
        }
        
        response = await self.client.post("/api/resource/Comment", json=data)
        response.raise_for_status()
        return response.json()["data"]
    
    async def get_customer_issues(
        self,
        customer: str,
        status: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Get all issues for a customer"""
        filters = [["customer", "=", customer]]
        if status:
            filters.append(["status", "=", status])
        
        response = await self.client.get(
            "/api/resource/Issue",
            params={
                "filters": str(filters),
                "fields": '["name", "subject", "status", "priority", "creation"]'
            }
        )
        response.raise_for_status()
        return response.json()["data"]
    
    async def create_customer_portal_user(
        self,
        email: str,
        first_name: str,
        last_name: str,
        customer: str
    ) -> Dict[str, Any]:
        """Create customer portal user"""
        data = {
            "doctype": "Contact",
            "email_id": email,
            "first_name": first_name,
            "last_name": last_name,
            "links": [{
                "link_doctype": "Customer",
                "link_name": customer
            }]
        }
        
        response = await self.client.post("/api/resource/Contact", json=data)
        response.raise_for_status()
        return response.json()["data"]
    
    async def get_customer_contracts(
        self,
        customer: str
    ) -> List[Dict[str, Any]]:
        """Get active contracts for customer"""
        filters = [
            ["party_name", "=", customer],
            ["status", "=", "Active"]
        ]
        
        response = await self.client.get(
            "/api/resource/Contract",
            params={
                "filters": str(filters),
                "fields": '["name", "contract_terms", "start_date", "end_date"]'
            }
        )
        response.raise_for_status()
        return response.json()["data"]
    
    async def get_customer_assets(
        self,
        customer: str
    ) -> List[Dict[str, Any]]:
        """Get assets for customer"""
        filters = [["customer", "=", customer]]
        
        response = await self.client.get(
            "/api/resource/Asset",
            params={
                "filters": str(filters),
                "fields": '["name", "asset_name", "item_code", "status"]'
            }
        )
        response.raise_for_status()
        return response.json()["data"]
    
    async def create_sales_invoice(
        self,
        customer: str,
        items: List[Dict[str, Any]],
        **kwargs
    ) -> Dict[str, Any]:
        """Create sales invoice (billing integration)"""
        data = {
            "doctype": "Sales Invoice",
            "customer": customer,
            "items": items,
            "posting_date": datetime.now().strftime("%Y-%m-%d"),
            **kwargs
        }
        
        response = await self.client.post("/api/resource/Sales Invoice", json=data)
        response.raise_for_status()
        return response.json()["data"]
    
    async def get_knowledge_base_articles(
        self,
        category: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Get knowledge base articles"""
        filters = []
        if category:
            filters.append(["category", "=", category])
        
        response = await self.client.get(
            "/api/resource/Help Article",
            params={
                "filters": str(filters) if filters else "[]",
                "fields": '["name", "title", "content", "category"]'
            }
        )
        response.raise_for_status()
        return response.json()["data"]
    
    async def close(self):
        """Close HTTP client"""
        await self.client.aclose()

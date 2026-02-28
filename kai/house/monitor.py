"""
KAI HOUSE - Infrastructure Health Monitor

Monitors infrastructure health, predicts capacity needs, and triggers auto-scaling.
"""

import logging
from typing import Dict, List, Optional
from dataclasses import dataclass
from datetime import datetime, timedelta
import statistics

logger = logging.getLogger(__name__)


@dataclass
class Alert:
    """Infrastructure alert"""
    severity: str  # critical, warning, info
    service: str
    message: str
    timestamp: datetime
    metric_name: str
    metric_value: float
    threshold: float


@dataclass
class CapacityPlan:
    """Capacity planning recommendation"""
    service: str
    current_replicas: int
    recommended_replicas: int
    reason: str
    confidence: float  # 0.0 to 1.0


class InfrastructureMonitor:
    """Monitors Kubric infrastructure health and capacity"""
    
    def __init__(self, docker_manager):
        self.docker_manager = docker_manager
        self.metric_history: Dict[str, List[Dict]] = {}
        self.alert_history: List[Alert] = []
    
    def detect_resource_exhaustion(self) -> List[Alert]:
        """
        Detect services approaching resource limits
        
        Returns:
            List of alerts for services with high resource usage
        """
        alerts = []
        
        try:
            usage = self.docker_manager.get_resource_usage()
            
            for service_name, metrics in usage.items():
                cpu_percent = metrics["cpu_percent"]
                memory_mb = metrics["memory_mb"]
                restart_count = metrics["restart_count"]
                
                # CPU exhaustion
                if cpu_percent > 90.0:
                    alerts.append(Alert(
                        severity="critical",
                        service=service_name,
                        message=f"CPU usage critical: {cpu_percent:.1f}%",
                        timestamp=datetime.now(),
                        metric_name="cpu_percent",
                        metric_value=cpu_percent,
                        threshold=90.0
                    ))
                elif cpu_percent > 80.0:
                    alerts.append(Alert(
                        severity="warning",
                        service=service_name,
                        message=f"CPU usage high: {cpu_percent:.1f}%",
                        timestamp=datetime.now(),
                        metric_name="cpu_percent",
                        metric_value=cpu_percent,
                        threshold=80.0
                    ))
                
                # Memory exhaustion
                if memory_mb > 900:  # Assuming 1GB limit
                    alerts.append(Alert(
                        severity="critical",
                        service=service_name,
                        message=f"Memory usage critical: {memory_mb:.1f} MB",
                        timestamp=datetime.now(),
                        metric_name="memory_mb",
                        metric_value=memory_mb,
                        threshold=900.0
                    ))
                elif memory_mb > 800:
                    alerts.append(Alert(
                        severity="warning",
                        service=service_name,
                        message=f"Memory usage high: {memory_mb:.1f} MB",
                        timestamp=datetime.now(),
                        metric_name="memory_mb",
                        metric_value=memory_mb,
                        threshold=800.0
                    ))
                
                # Restart loop detection
                if restart_count > 5:
                    alerts.append(Alert(
                        severity="critical",
                        service=service_name,
                        message=f"Service in restart loop: {restart_count} restarts",
                        timestamp=datetime.now(),
                        metric_name="restart_count",
                        metric_value=float(restart_count),
                        threshold=5.0
                    ))
                elif restart_count > 3:
                    alerts.append(Alert(
                        severity="warning",
                        service=service_name,
                        message=f"Service restarting frequently: {restart_count} restarts",
                        timestamp=datetime.now(),
                        metric_name="restart_count",
                        metric_value=float(restart_count),
                        threshold=3.0
                    ))
            
            # Store alerts
            self.alert_history.extend(alerts)
            
            # Keep only last 1000 alerts
            if len(self.alert_history) > 1000:
                self.alert_history = self.alert_history[-1000:]
            
        except Exception as e:
            logger.error(f"Failed to detect resource exhaustion: {e}")
        
        return alerts
    
    def collect_metrics(self):
        """Collect and store metrics for trend analysis"""
        try:
            usage = self.docker_manager.get_resource_usage()
            timestamp = datetime.now()
            
            for service_name, metrics in usage.items():
                if service_name not in self.metric_history:
                    self.metric_history[service_name] = []
                
                self.metric_history[service_name].append({
                    "timestamp": timestamp,
                    "cpu_percent": metrics["cpu_percent"],
                    "memory_mb": metrics["memory_mb"],
                    "restart_count": metrics["restart_count"]
                })
                
                # Keep only last 24 hours of metrics (assuming 1 min interval = 1440 points)
                if len(self.metric_history[service_name]) > 1440:
                    self.metric_history[service_name] = self.metric_history[service_name][-1440:]
            
        except Exception as e:
            logger.error(f"Failed to collect metrics: {e}")
    
    def predict_capacity_needs(self) -> List[CapacityPlan]:
        """
        Predict capacity needs based on historical trends
        
        Returns:
            List of capacity planning recommendations
        """
        plans = []
        
        try:
            health = self.docker_manager.get_service_health()
            
            for service_name, service_health in health.items():
                if service_name not in self.metric_history:
                    continue
                
                history = self.metric_history[service_name]
                
                if len(history) < 10:
                    # Not enough data
                    continue
                
                # Get recent metrics (last hour)
                recent_metrics = [m for m in history if m["timestamp"] > datetime.now() - timedelta(hours=1)]
                
                if not recent_metrics:
                    continue
                
                # Calculate average CPU and memory
                avg_cpu = statistics.mean([m["cpu_percent"] for m in recent_metrics])
                avg_memory = statistics.mean([m["memory_mb"] for m in recent_metrics])
                
                # Calculate trend (simple linear regression)
                cpu_values = [m["cpu_percent"] for m in recent_metrics]
                cpu_trend = (cpu_values[-1] - cpu_values[0]) / len(cpu_values) if len(cpu_values) > 1 else 0
                
                current_replicas = service_health.replicas
                recommended_replicas = current_replicas
                reason = ""
                confidence = 0.0
                
                # Scale up if trending upward and high usage
                if cpu_trend > 5.0 and avg_cpu > 70.0:
                    recommended_replicas = current_replicas + 1
                    reason = f"CPU trending up ({cpu_trend:.1f}%/min) with high average ({avg_cpu:.1f}%)"
                    confidence = min(1.0, (avg_cpu - 70.0) / 30.0)
                
                # Scale down if consistently low usage
                elif avg_cpu < 30.0 and avg_memory < 300 and current_replicas > 1:
                    recommended_replicas = max(1, current_replicas - 1)
                    reason = f"Low resource usage (CPU: {avg_cpu:.1f}%, Memory: {avg_memory:.1f} MB)"
                    confidence = min(1.0, (30.0 - avg_cpu) / 30.0)
                
                if recommended_replicas != current_replicas:
                    plans.append(CapacityPlan(
                        service=service_name,
                        current_replicas=current_replicas,
                        recommended_replicas=recommended_replicas,
                        reason=reason,
                        confidence=confidence
                    ))
            
        except Exception as e:
            logger.error(f"Failed to predict capacity needs: {e}")
        
        return plans
    
    def auto_scale_decision(self) -> Dict[str, int]:
        """
        Make auto-scaling decisions based on current state and predictions
        
        Returns:
            Dict mapping service name to target replica count
        """
        decisions = {}
        
        try:
            # Get immediate alerts
            alerts = self.detect_resource_exhaustion()
            
            # Get capacity predictions
            plans = self.predict_capacity_needs()
            
            # Prioritize critical alerts
            for alert in alerts:
                if alert.severity == "critical" and alert.metric_name in ["cpu_percent", "memory_mb"]:
                    service_name = alert.service
                    health = self.docker_manager.get_service_health(service_name)
                    
                    if health:
                        service_health = list(health.values())[0]
                        current_replicas = service_health.replicas
                        decisions[service_name] = current_replicas + 1
                        logger.info(f"Auto-scale decision: {service_name} -> {current_replicas + 1} replicas (critical alert)")
            
            # Apply high-confidence capacity plans
            for plan in plans:
                if plan.confidence > 0.7 and plan.service not in decisions:
                    decisions[plan.service] = plan.recommended_replicas
                    logger.info(f"Auto-scale decision: {plan.service} -> {plan.recommended_replicas} replicas ({plan.reason})")
            
        except Exception as e:
            logger.error(f"Failed to make auto-scale decision: {e}")
        
        return decisions
    
    def get_health_summary(self) -> Dict:
        """
        Get overall infrastructure health summary
        
        Returns:
            Dict with health metrics and status
        """
        summary = {
            "timestamp": datetime.now().isoformat(),
            "overall_status": "healthy",
            "services": {},
            "alerts": {
                "critical": 0,
                "warning": 0,
                "info": 0
            },
            "capacity_recommendations": []
        }
        
        try:
            # Get service health
            health = self.docker_manager.get_service_health()
            
            for service_name, service_health in health.items():
                summary["services"][service_name] = {
                    "status": service_health.status,
                    "health": service_health.health,
                    "cpu_percent": service_health.cpu_percent,
                    "memory_mb": service_health.memory_mb,
                    "restart_count": service_health.restart_count
                }
                
                if service_health.health != "healthy":
                    summary["overall_status"] = "degraded"
            
            # Get recent alerts
            recent_alerts = [a for a in self.alert_history if a.timestamp > datetime.now() - timedelta(minutes=5)]
            
            for alert in recent_alerts:
                summary["alerts"][alert.severity] += 1
            
            if summary["alerts"]["critical"] > 0:
                summary["overall_status"] = "critical"
            elif summary["alerts"]["warning"] > 0 and summary["overall_status"] == "healthy":
                summary["overall_status"] = "warning"
            
            # Get capacity recommendations
            plans = self.predict_capacity_needs()
            summary["capacity_recommendations"] = [
                {
                    "service": p.service,
                    "current_replicas": p.current_replicas,
                    "recommended_replicas": p.recommended_replicas,
                    "reason": p.reason,
                    "confidence": p.confidence
                }
                for p in plans
            ]
            
        except Exception as e:
            logger.error(f"Failed to get health summary: {e}")
            summary["overall_status"] = "unknown"
            summary["error"] = str(e)
        
        return summary

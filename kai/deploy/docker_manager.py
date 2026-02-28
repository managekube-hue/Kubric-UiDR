"""
KAI DEPLOY - Docker Infrastructure Manager

Enables AI-driven deployment, scaling, and remediation of Kubric services.
"""

import docker
import logging
from typing import Dict, List, Optional
from dataclasses import dataclass
from datetime import datetime

logger = logging.getLogger(__name__)


@dataclass
class ServiceHealth:
    """Service health status"""
    name: str
    status: str  # running, restarting, exited
    health: str  # healthy, unhealthy, starting
    replicas: int
    cpu_percent: float
    memory_mb: float
    restart_count: int


class DockerManager:
    """Manages Docker services for Kubric platform"""
    
    def __init__(self):
        try:
            self.client = docker.from_env()
            logger.info("Docker client initialized")
        except Exception as e:
            logger.error(f"Failed to initialize Docker client: {e}")
            raise
    
    def get_service_health(self, service_name: Optional[str] = None) -> Dict[str, ServiceHealth]:
        """
        Get health status of services
        
        Args:
            service_name: Specific service name, or None for all services
            
        Returns:
            Dict mapping service name to ServiceHealth
        """
        health_map = {}
        
        try:
            filters = {"label": "com.docker.compose.project=kubric-uidr"}
            if service_name:
                filters["name"] = service_name
            
            containers = self.client.containers.list(all=True, filters=filters)
            
            for container in containers:
                name = container.name
                stats = container.stats(stream=False)
                
                # Calculate CPU percentage
                cpu_delta = stats["cpu_stats"]["cpu_usage"]["total_usage"] - \
                           stats["precpu_stats"]["cpu_usage"]["total_usage"]
                system_delta = stats["cpu_stats"]["system_cpu_usage"] - \
                              stats["precpu_stats"]["system_cpu_usage"]
                cpu_percent = (cpu_delta / system_delta) * 100.0 if system_delta > 0 else 0.0
                
                # Calculate memory usage
                memory_mb = stats["memory_stats"]["usage"] / (1024 * 1024)
                
                # Get restart count
                restart_count = container.attrs["RestartCount"]
                
                # Determine health
                health_status = "unknown"
                if "Health" in container.attrs["State"]:
                    health_status = container.attrs["State"]["Health"]["Status"]
                elif container.status == "running":
                    health_status = "healthy"
                
                health_map[name] = ServiceHealth(
                    name=name,
                    status=container.status,
                    health=health_status,
                    replicas=1,  # TODO: Get from compose scale
                    cpu_percent=cpu_percent,
                    memory_mb=memory_mb,
                    restart_count=restart_count
                )
                
        except Exception as e:
            logger.error(f"Failed to get service health: {e}")
        
        return health_map
    
    def scale_service(self, service_name: str, replicas: int) -> bool:
        """
        Scale service to N replicas
        
        Args:
            service_name: Name of service to scale
            replicas: Target replica count
            
        Returns:
            True if successful
        """
        try:
            # Use docker-compose scale command
            import subprocess
            result = subprocess.run(
                ["docker", "compose", "-f", "docker-compose.prod.yml", 
                 "up", "-d", "--scale", f"{service_name}={replicas}", "--no-recreate"],
                capture_output=True,
                text=True
            )
            
            if result.returncode == 0:
                logger.info(f"Scaled {service_name} to {replicas} replicas")
                return True
            else:
                logger.error(f"Failed to scale {service_name}: {result.stderr}")
                return False
                
        except Exception as e:
            logger.error(f"Failed to scale service {service_name}: {e}")
            return False
    
    def restart_service(self, service_name: str) -> bool:
        """
        Restart a service
        
        Args:
            service_name: Name of service to restart
            
        Returns:
            True if successful
        """
        try:
            containers = self.client.containers.list(
                filters={"name": service_name, "label": "com.docker.compose.project=kubric-uidr"}
            )
            
            for container in containers:
                container.restart(timeout=10)
                logger.info(f"Restarted container {container.name}")
            
            return True
            
        except Exception as e:
            logger.error(f"Failed to restart service {service_name}: {e}")
            return False
    
    def rollback_service(self, service_name: str) -> bool:
        """
        Rollback service to previous image version
        
        Args:
            service_name: Name of service to rollback
            
        Returns:
            True if successful
        """
        try:
            # Get current container
            containers = self.client.containers.list(
                filters={"name": service_name, "label": "com.docker.compose.project=kubric-uidr"}
            )
            
            if not containers:
                logger.error(f"No containers found for service {service_name}")
                return False
            
            container = containers[0]
            current_image = container.image.tags[0] if container.image.tags else None
            
            if not current_image:
                logger.error(f"Cannot determine current image for {service_name}")
                return False
            
            # Parse image name and tag
            image_name, current_tag = current_image.rsplit(":", 1)
            
            # Get previous tag (simple version: assume previous is one version back)
            # In production, this should query a registry or maintain version history
            previous_tag = "previous"  # Placeholder
            previous_image = f"{image_name}:{previous_tag}"
            
            logger.info(f"Rolling back {service_name} from {current_image} to {previous_image}")
            
            # Pull previous image
            self.client.images.pull(previous_image)
            
            # Recreate container with previous image
            import subprocess
            result = subprocess.run(
                ["docker", "compose", "-f", "docker-compose.prod.yml", 
                 "up", "-d", "--no-deps", "--force-recreate", service_name],
                capture_output=True,
                text=True,
                env={"KUBRIC_IMAGE_TAG": previous_tag}
            )
            
            if result.returncode == 0:
                logger.info(f"Rolled back {service_name} successfully")
                return True
            else:
                logger.error(f"Failed to rollback {service_name}: {result.stderr}")
                return False
                
        except Exception as e:
            logger.error(f"Failed to rollback service {service_name}: {e}")
            return False
    
    def auto_remediate(self, service_name: str) -> bool:
        """
        Auto-remediate unhealthy service
        
        Decision tree:
        1. If restart_count < 3: restart
        2. If restart_count >= 3 and < 5: scale up
        3. If restart_count >= 5: rollback
        
        Args:
            service_name: Name of service to remediate
            
        Returns:
            True if remediation attempted
        """
        try:
            health = self.get_service_health(service_name)
            
            if not health:
                logger.error(f"Cannot get health for {service_name}")
                return False
            
            service_health = list(health.values())[0]
            
            if service_health.health == "healthy":
                logger.info(f"{service_name} is healthy, no remediation needed")
                return True
            
            restart_count = service_health.restart_count
            
            if restart_count < 3:
                logger.info(f"Restarting {service_name} (restart_count={restart_count})")
                return self.restart_service(service_name)
            
            elif restart_count < 5:
                logger.info(f"Scaling up {service_name} (restart_count={restart_count})")
                current_replicas = service_health.replicas
                return self.scale_service(service_name, current_replicas + 1)
            
            else:
                logger.info(f"Rolling back {service_name} (restart_count={restart_count})")
                return self.rollback_service(service_name)
                
        except Exception as e:
            logger.error(f"Failed to auto-remediate {service_name}: {e}")
            return False
    
    def get_resource_usage(self) -> Dict[str, Dict[str, float]]:
        """
        Get resource usage for all services
        
        Returns:
            Dict mapping service name to resource metrics
        """
        usage = {}
        
        try:
            health = self.get_service_health()
            
            for name, service_health in health.items():
                usage[name] = {
                    "cpu_percent": service_health.cpu_percent,
                    "memory_mb": service_health.memory_mb,
                    "restart_count": service_health.restart_count
                }
                
        except Exception as e:
            logger.error(f"Failed to get resource usage: {e}")
        
        return usage
    
    def predict_scale_needs(self) -> List[str]:
        """
        Predict which services need scaling based on resource usage
        
        Returns:
            List of service names that should be scaled up
        """
        scale_candidates = []
        
        try:
            health = self.get_service_health()
            
            for name, service_health in health.items():
                # Scale up if CPU > 80% or memory > 85%
                if service_health.cpu_percent > 80.0:
                    logger.warning(f"{name} CPU usage high: {service_health.cpu_percent:.1f}%")
                    scale_candidates.append(name)
                
                elif service_health.memory_mb > (service_health.memory_mb * 0.85):
                    logger.warning(f"{name} memory usage high: {service_health.memory_mb:.1f} MB")
                    scale_candidates.append(name)
                    
        except Exception as e:
            logger.error(f"Failed to predict scale needs: {e}")
        
        return scale_candidates

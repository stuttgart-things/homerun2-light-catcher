"""
Main entry point for WLED mock server Kubernetes deployment
Imports all resources and exports manifests
"""

import namespace
import serviceaccount
import deploy
import service
import httproute

# Export all manifests (filter out None values)
manifests = [
    m for m in [
        namespace.namespace
        serviceaccount.serviceAccount
        deploy.deployment
        service.service
        httproute.httproute
    ] if m
]

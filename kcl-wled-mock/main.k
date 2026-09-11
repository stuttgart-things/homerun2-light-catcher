"""
Main entry point for WLED mock server Kubernetes deployment
Imports all resources and exports manifests
"""

import serviceaccount
import deploy
import service
import httproute

# Export all manifests (filter out None values).
# Namespace is NOT emitted from the KCL profile — the consuming Application
# uses `CreateNamespace=true`, and PR-preview namespaces are pre-created by
# Kyverno. See stuttgart-things/homerun2-light-catcher#35.
manifests = [
    m for m in [
        serviceaccount.serviceAccount
        deploy.deployment
        service.service
        httproute.httproute
    ] if m
]

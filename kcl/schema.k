"""
Schema definition for homerun2-light-catcher deployment configuration
"""

schema LightCatcher:
    """Configuration for homerun2-light-catcher Kubernetes deployment"""

    # Metadata
    name: str = "homerun2-light-catcher"
    namespace: str = "homerun2"

    # Image
    image: str = "ghcr.io/stuttgart-things/homerun2-light-catcher:latest"
    imagePullPolicy: str = "IfNotPresent"

    # Replicas and resources
    replicas: int = 1
    cpuRequest: str = "50m"
    cpuLimit: str = "200m"
    memoryRequest: str = "64Mi"
    memoryLimit: str = "256Mi"

    # Redis
    redisAddr: str = "redis-stack.homerun2.svc.cluster.local"
    redisPort: int | str = "6379"
    redisStream: str = "messages"

    # Consumer group
    consumerGroup: str = "homerun2-light-catcher"

    # Health endpoint
    healthPort: int | str = "8080"

    # Service
    serviceType: str = "ClusterIP"
    servicePort: int = 80
    containerPort: int = 8080

    # HTTPRoute (Gateway API)
    httpRouteEnabled: bool = False
    httpRouteParentRefName: str = ""
    httpRouteParentRefNamespace: str = ""
    httpRouteHostname: str = ""
    httpRouteAnnotations: {str:str} = {}

    # Secrets
    redisPassword: str = ""

    # Profile path inside container
    profilePath: str = "/config/profile.yaml"

    # Profile data (WLED effect rules YAML content). Rendered into the
    # `<name>-profile` ConfigMap that the Deployment mounts at /config.
    # Default endpoint targets `homerun2-wled-mock` so the kustomize OCI
    # is self-contained against the wled-mock kustomize OCI's Service
    # (same component pair as upstream; deployable side-by-side in a
    # single namespace).
    profileData: str = """\
---
effects:
  error:
    systems:
      - "*"
    severity:
      - error
      - critical
    fx: Blurz
    duration: 3
    color: sunset
    segments:
      - 0
    endpoint: http://homerun2-wled-mock

  warning:
    systems:
      - "*"
    severity:
      - warning
    fx: Twinkle
    duration: 3
    color: beach
    segments:
      - 0
    endpoint: http://homerun2-wled-mock

  success:
    systems:
      - "*"
    severity:
      - success
    fx: Aurora
    duration: 3
    color: forest
    segments:
      - 0
    endpoint: http://homerun2-wled-mock

  info:
    systems:
      - "*"
    severity:
      - info
    fx: DJ Light
    duration: 3
    color: ocean
    segments:
      - 0
    endpoint: http://homerun2-wled-mock
"""

    # Extra environment variables
    extraEnvVars: {str:str} = {}

    # Labels and annotations
    labels: {str:str} = {}
    annotations: {str:str} = {}

    check:
        replicas >= 0, "replicas must be >= 0"
        servicePort >= 1 and servicePort <= 65535, "servicePort must be valid"
        containerPort >= 1 and containerPort <= 65535, "containerPort must be valid"
        name != "", "name must not be empty"
        namespace != "", "namespace must not be empty"

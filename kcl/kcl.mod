[package]
name = "deploy-homerun2-light-catcher"
version = "0.1.0"
description = "KCL module for deploying homerun2-light-catcher on Kubernetes"

[dependencies]
k8s = "1.36"

[profile]
entries = [
    "main.k"
]

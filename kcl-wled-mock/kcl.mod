[package]
name = "deploy-homerun2-wled-mock"
version = "0.1.0"
description = "KCL module for deploying the WLED mock server on Kubernetes"

[dependencies]
k8s = "1.31"

[profile]
entries = [
    "main.k"
]

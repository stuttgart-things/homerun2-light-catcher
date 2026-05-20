"""
Profile ConfigMap for homerun2-light-catcher
Contains the WLED effect rules YAML mounted at /config/profile.yaml
"""

import labels

config = labels.config

profileConfigMap = {
    apiVersion: "v1"
    kind: "ConfigMap"
    metadata: {
        name: "{}-profile".format(config.name)
        namespace: config.namespace
        labels: labels.commonLabels
    }
    data: {
        "profile.yaml": config.profileData
    }
}

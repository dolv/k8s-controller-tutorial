# app Helm Chart

This Helm chart deploys the app with configurable image repository and tag.

## Usage

Override the image tag to deploy a specific version:

```sh
helm upgrade --install jaegernginxproxy-controller ./charts/app \
  --set image.tag=0.1.0-0aff3707
```

The image tag is set by CI to the Git tag (if present) or the commit SHA.
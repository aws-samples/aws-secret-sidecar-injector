# AWS Secret Sidecar Injector

The _aws-secret-sidecar-injector_ is a proof-of-concept(PoC) that allows your containerized applications to consume secrets from AWS Secrets Manager. The solution makes use of a Kubernetes dynamic admission controller that injects an _init_ container, aws-secrets-manager-secret-sidecar, upon creation/update of your pod. The init container relies on [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) to retrieve the secret from AWS Secrets Manager. The Kubernetes dynamic admission controller also creates an in-memory Kubernetes volume (with name `secret-vol` and `emptyDirectory.medium` as `Memory`) associated with the pod to access the secret.

## Prerequsites 
- An IRSA ServiceAccount that has permission to access and retrive the secret from AWS Secrets Manager
- Helm to install the mutating admission webhook

## Installation

### Deploying mutatating webhook to inject the init container 

- Add the Helm repository which contains the Helm chart for the mutating admission webhook 

  ```helm repo add secret-inject https://aws-samples.github.io/aws-secret-sidecar-injector/```

- Update the Helm repository 

  ```helm repo update```

- Deploy the mutating webhook admission controller

  ```helm install secret-inject secret-inject/secret-inject```

## Accessing the secret

Add the following annotations to your podSpec to mount the secret in your pod 

  ```secrets.k8s.aws/sidecarInjectorWebhook: enabled```

  ```secrets.k8s.aws/secret-arn: <SECRET-ARN>```
  
The decrypted secret is written to a volume named `secret-vol` and the filename of the secret is `secret`. The Kubernetes dynamic admission controller also creates corresponding mountPath `/tmp/secret` for containers within the pod to access the secret. 

This repository contains a sample Kubernetes deployment [manifest](https://github.com/aws-samples/aws-secret-sidecar-injector/blob/master/kubernetes-manifests/webserver.yaml) which uses this project to access AWS Secrets Manager secret.  

## Creating Secrets

AWS Secrets Manager secrets can be created and managed natively in Kubernetes using [Native Secrets(NASE)](https://github.com/mhausenblas/nase). The NASE project is a serverless mutating webhook, which "intercepts" the calls to create and update native Kubernetes Secrets and writes the secret in the secret manifest to AWS Secrets Manager and returns the ARN of the secret to Kubernetes which stores it as a secret.

## License

This library is licensed under the MIT-0 License. See the LICENSE file.


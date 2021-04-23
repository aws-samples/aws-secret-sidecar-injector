# AWS Secret Sidecar Injector

The _aws-secret-sidecar-injector_ is a proof-of-concept(PoC) that allows your containerized applications to consume secrets from AWS Secrets Manager. The solution makes use of a Kubernetes dynamic admission controller that injects an _init_ container, aws-secrets-manager-secret-sidecar, upon creation/update of your pod. The init container relies on [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) to retrieve the secret from AWS Secrets Manager. The Kubernetes dynamic admission controller also creates an in-memory Kubernetes volume (with name `secret-vol` and `emptyDirectory.medium` as `Memory`) associated with the pod to access the secret.

## Announcing the AWS Secrets and Config Provider (ASCP)

As of 4/22/21, you use the CSI Secret Store driver with AWS Secrets Manager and Parameter Store. ASCP is similar to this project in that it mounts secrets as volumes, however there are several key differences that are worth highlighting. First, it works with both Secrets Manager **and** Parameter store. Second, ASCP can mount multiple secrets whereas the sidecar injector only supports 1. Third, ASCP can synchronize secrets from Secrets Manager to Kubernetes Secrets; this is similar to GoDaddy's [ExternalSecrets](https://github.com/external-secrets/kubernetes-external-secrets) project. By copying secrets from Secrets Manager to Kubernetes Secrets you can map them to environment variables instead of mounting them as volumes. Fourth, ASCP can rotate secrets, however, unlike the sidecar injector, ASCP uses a polling mechanism rather than an event to trigger the rotation. When we were thinking of how to handle the rotation of secrets we decided to use an event rather than polling to a) limit the resources required to continuously run the sidecar and b) to keep costs low; Secrets Manager charges $0.05 per 10,000 API calls. Fifth, with ASCP you have to create a secret provider class for each secret you want to reference in your pod. 

> You can still use Michael Hausenblas's [NASE](https://github.com/mhausenblas/nase) project to create secrets in Secrets Manager. 

We will continue supporting this project, but we also encourage you to give ASCP a try. Thank you to all of those who provided feedback and helped make this project what it is today. For additional information about ASCP see: 

+ [How to use AWS Secrets Configuration Provider with Kubernetes Secret Store CSI Driver](https://aws.amazon.com/blogs/security/how-to-use-aws-secrets-configuration-provider-with-kubernetes-secrets-store-csi-driver/)
+ [Secret Store CSI Driver Provider AWS](https://github.com/aws/secrets-store-csi-driver-provider-aws)
+ [Integrating CSI Driver](https://docs.aws.amazon.com/secretsmanager/latest/userguide/integrating_csi_driver.html)

## Prerequsites 
- An IRSA ServiceAccount that has permission to access and retrive the secret from AWS Secrets Manager
- Helm to install the mutating admission webhook

## Installation

### Deploying mutating webhook to inject the init container 

- Add the Helm repository which contains the Helm chart for the mutating admission webhook 

  ```helm repo add secret-inject https://aws-samples.github.io/aws-secret-sidecar-injector/```

- Update the Helm repository 

  ```helm repo update```

- Deploy the mutating webhook admission controller

  ```helm install secret-inject secret-inject/secret-inject```

## Accessing the secret

Add the following annotations to your podSpec to mount the secret in your pod 

  ```secrets.k8s.aws/secret-arn: <SECRET-ARN>```
  
By default, the decrypted secret is written to a volume named `secret-vol` and the filename of the secret is `secret`. The Kubernetes dynamic admission controller also creates corresponding mountPath `/tmp/secret` for containers within the pod to access the secret.

You can optionally mount the `secret-vol` volume for containers within the pod at a specific path using the following optional annotation

  ```secrets.k8s.aws/mount-path: <ABSOULTE-MOUNT-PATH>```
  
Note that,the path should be an absolute path such as "/my-path"
  
You can optionally customize the filename / subfolders within the mounted path where the secret is written by using hte following optional annotation

   ```secrets.k8s.aws/secret-filename: <SECRET-FILENAME>```
   
This repository contains a sample Kubernetes deployment [manifest](https://github.com/aws-samples/aws-secret-sidecar-injector/blob/master/kubernetes-manifests/webserver.yaml) which uses this project to access AWS Secrets Manager secret.  

## Creating Secrets

AWS Secrets Manager secrets can be created and managed natively in Kubernetes using [Native Secrets(NASE)](https://github.com/mhausenblas/nase). The NASE project is a serverless mutating webhook, which "intercepts" the calls to create and update native Kubernetes Secrets and writes the secret in the secret manifest to AWS Secrets Manager and returns the ARN of the secret to Kubernetes which stores it as a secret.

## Rotating Secrets

Support for restarting pods when the secret they reference is rotated, is now available.  For additional information, see the [README](https://github.com/aws-samples/aws-secret-sidecar-injector/blob/master/secret-operator/README.md) in the secret-operator folder. 

## License

This library is licensed under the MIT-0 License. See the LICENSE file.


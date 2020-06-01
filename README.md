## AWS Secret Sidecar Injector

The aws-secret-sidecar-inject is a proof-of-concept(PoC) project to retrieve secret from AWS Secrets Manager and access it in your containerized application. This project mutates your application pod to inject an init container(secret-inject-init) upon creation / update of your pod. The init container uses [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) privileges to retrieve the secret from AWS Secrets Manager. The secret is mounted as a RAM disk persistent volume (emptyDirectory.medium as memory) to be shared with the application container. This repository also contains a mutating admisssion webhook controller which injects the init container upon detecting the following annotations in the pod spec
- secrets.k8s.aws/sidecarInjectorWebhook:enabled
- secrets.k8s.aws/secret-arn: <secret-ARN>

### Prerequsites 
- IRSA to access and retrive the secret from AWS Secrets Manager
- Helm to install the mutating admission webhook

### Installation

#### Deploying mutatating webhook to inject the init container 

- Add the Helm repository which contains the Helm chart for the mutating admission webhook 

```helm repo add secret-inject http://aws-samples.github.io/aws-secret-sidecar-injector/```
- Update the Helm repository 

```helm repo update```
- Deploy the mutating webhook admission controller

```helm install secret-inject secret-inject/secret-inject```

### Accessing the secret

Add the following annotations to your pod spec to access the secret in your pod 
```secrets.k8s.aws/sidecarInjectorWebhook:enabled```

```secrets.k8s.aws/secret-arn: <SECRET-ARN>```

## License

This library is licensed under the MIT-0 License. See the LICENSE file.


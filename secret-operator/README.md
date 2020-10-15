# Introduction
This project helps users to automatically redeploy the pods running on Amazon EKS cluster when the secrets in AWS Secerets Manager is rotated. When the pods are restarted, webhook in our previous [blog](https://aws.amazon.com/blogs/containers/aws-secrets-controller-poc/) will retrive the latest secret and mount it onto the pods.

![GitHub Logo](blog3.jpg)

## Install the CRD and Operator
1. kubectl should be configured to acces EKS cluster on the system where you build the project - https://docs.aws.amazon.com/eks/latest/userguide/install-kubectl.html

2. Install kubebuilder - https://book.kubebuilder.io/quick-start.html#installation

3. To test the operator we need a SQS queue and AWS EventBridge rule, which will store the event details of the PutSecretValue API call, so that the Secrets controller can get the secret rotation details. You can either use existing resources or create resources by run the below command -
```
make aws
```
Running the above command will create a CloudFormation stack provisining following resources -
```
* sample secret
* EventBridge rule
* IAM role for operator IRSA
* SQS queue
```

4. Clone the project into go project path -   
```
cd ~/go/src && git clone https://github.com/Mahendrasiddappa/secretoperator.git && cd secretoperator
```

5. Following commands will get region, SQS URL and IRSA IAM role arn from the CloudFormation stack created in step 3. If you want to use existing resources in your account you can pass those vaules to the below variables - 
* ```export OPERATOR_REGION=$(aws cloudformation describe-stacks --stack-name EKS-Secrets-Operator-Stack --query "Stacks[0].Outputs[?OutputKey=='Region'].OutputValue" --output text)```
* ```export SQS_URL=$(aws cloudformation describe-stacks --stack-name EKS-Secrets-Operator-Stack --query "Stacks[0].Outputs[?OutputKey=='QueueURL'].OutputValue" --output text)```
* ```export IAM_ARN=$(aws cloudformation describe-stacks --stack-name EKS-Secrets-Operator-Stack --query "Stacks[0].Outputs[?OutputKey=='IAMRole'].OutputValue" --output text)```

6. Replace those values in the controller deployment configuration - 
* ```sed -i "s,SQS_URL,${SQS_URL},g" config/manager/manager.yaml```
* ```sed -i "s,OPERATOR_REGION,${OPERATOR_REGION},g" config/manager/manager.yaml```
* ```sed -i "s,IAM_ARN,${IAM_ARN},g" config/manager/manager.yaml```

7. Install CRD -   
```
make install
```

7. Build and push the controller image to your repository -   
```
make docker-build docker-push IMG=<registry>:<tag>
```

8. deploy the controller on the cluster 
```
make deploy IMG=<registry>:<tag>
```


## Testing 
1. Create CRD in default namespace which will look for Deployments, Daemonsets and Statefulset's with labesl "environment: OperatorTest" -
  ```
  kubectl create -f config/samples/awssecretsoperator_v1_secretsrotationmapping.yaml
  ```

2. Create a deployment which runs nginx pods and has labels "environment: operatortest" - 
  ```
  kubectl create -f config/samples/deployment.yaml
  ```

3. Create PutSecretValue event -
```
aws secretsmanager put-secret-value --secret-id eks-controller-test-secret --secret-string [{testsqssec:newsecret}]
```

## Result - 
The secrets-nginx deployment should restart the pods


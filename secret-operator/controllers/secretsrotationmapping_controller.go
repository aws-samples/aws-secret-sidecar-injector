/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-logr/logr"
	"k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	awssecretsoperatorv1 "secretoperator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// SecretsRotationMappingReconciler reconciles a SecretsRotationMapping object
type SecretsRotationMappingReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	RequeueAfter time.Duration
	QueueUrl     string
	Region       string
}

// +kubebuilder:rbac:groups=awssecretsoperator.secretoperator,resources=secretsrotationmappings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=awssecretsoperator.secretoperator,resources=secretsrotationmappings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=list;watch;create;update;patch;delete

func (r *SecretsRotationMappingReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	var DeleteMessageBatchList []*sqs.DeleteMessageBatchRequestEntry
	var SecretsRotationMapping awssecretsoperatorv1.SecretsRotationMapping
	var result map[string]interface{}

	if err := r.Get(ctx, req.NamespacedName, &SecretsRotationMapping); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//fmt.Println(SecretsRotationMapping.Spec.Labels)
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(r.Region)},
	)
	svc := sqs.New(sess)

	//read message from SQS
	message, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            &r.QueueUrl,
		MaxNumberOfMessages: aws.Int64(10),
		VisibilityTimeout:   aws.Int64(2),
		WaitTimeSeconds:     aws.Int64(0),
	})

	if err != nil {
		fmt.Println("Error", err)
		return ctrl.Result{RequeueAfter: time.Second * r.RequeueAfter}, nil
	}

	//fmt.Println("SQS messages:", message.Messages)
	//loop through all the messages retrived from SQS
	for _, element := range message.Messages {
		err := json.Unmarshal([]byte(*element.Body), &result)
		if err != nil {
			fmt.Println("Error", err)
		}

		detail := result["detail"].(map[string]interface{})
		eventName := detail["eventName"]

		// continue only if the event type is PutSecretValue
		if eventName == "PutSecretValue" {
			requestParameters := detail["requestParameters"].(map[string]interface{})
			secretID := requestParameters["secretId"]
			fmt.Println("Secret ID rotated", secretID)

			//if the secretID in SQS message is not same as the secret in CRD, continue with next message
			if secretID != SecretsRotationMapping.Spec.SecretID {
				fmt.Println("continuing to next loop")
				continue
			}

			//get the deployment using labesl specified in the crd SecretsRotationMapping
			var deploy v1.DeploymentList
			//MatchingLabels := SecretsRotationMapping.Spec.Labels
			r.List(ctx, &deploy, client.MatchingLabels(SecretsRotationMapping.Spec.Labels))
			//	fmt.Println("List deployments by Label:", deploy)

			for _, deployment := range deploy.Items {
				// Patch the Deployment with new label containing redeployed timestamp, to force redeploy
				fmt.Println("Rotating deployment", deployment.ObjectMeta.Name)
				patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"labels":{"aws-secrets-controller-redeloyed":"%v"}}}}}`, time.Now().Unix()))
				if err := r.Patch(ctx, &deployment, client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
					fmt.Println("Patch deployment err:", err)
					return ctrl.Result{RequeueAfter: time.Second * r.RequeueAfter}, nil
				}
			}

			//get the DaemonSet using labesl specified in the crd SecretsRotationMapping
			var DaemonSetList v1.DaemonSetList
			r.List(ctx, &DaemonSetList, client.MatchingLabels(SecretsRotationMapping.Spec.Labels))
			//	fmt.Println("List DaemonSetList by Label:", DaemonSetList)

			for _, DaemonSet := range DaemonSetList.Items {
				// Patch the DaemonSet with new label containing redeployed timestamp, to force redeploy
				fmt.Println("Rotating DaemonSet", DaemonSet.ObjectMeta.Name)
				patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"labels":{"aws-secrets-operator-redeloyed":"%v"}}}}}`, time.Now().Unix()))
				if err := r.Patch(ctx, &DaemonSet, client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
					fmt.Println("Patch DaemonSet err:", err)
					return ctrl.Result{RequeueAfter: time.Second * r.RequeueAfter}, nil
				}
			}

			//get the SatefulSet using labesl specified in the crd SecretsRotationMapping
			var StatefulSetList v1.StatefulSetList
			r.List(ctx, &StatefulSetList, client.MatchingLabels(SecretsRotationMapping.Spec.Labels))

			for _, StatefulSet := range StatefulSetList.Items {
				// Patch the StatefulSet with new label containing redeployed timestamp, to force redeploy
				fmt.Println("Rotating StatefulSet", StatefulSet.ObjectMeta.Name)
				patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"labels":{"aws-secrets-operator-redeloyed":"%v"}}}}}`, time.Now().Unix()))
				if err := r.Patch(ctx, &StatefulSet, client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
					fmt.Println("Patch StatefulSet err:", err)
					return ctrl.Result{RequeueAfter: time.Second * r.RequeueAfter}, nil
				}
			}
		}

		deleteMessage := sqs.DeleteMessageBatchRequestEntry{Id: element.MessageId, ReceiptHandle: element.ReceiptHandle}
		DeleteMessageBatchList = append(DeleteMessageBatchList, &deleteMessage)
	}

	//DeleteMessageBatch
	//fmt.Println("DeleteMessageBatchList:", DeleteMessageBatchList)
	if len(DeleteMessageBatchList) > 0 {
		DeleteMessageBatchInput := &sqs.DeleteMessageBatchInput{Entries: DeleteMessageBatchList, QueueUrl: &r.QueueUrl}
		DeleteMessageBatchOutput, err := svc.DeleteMessageBatch(DeleteMessageBatchInput)
		if err != nil {
			fmt.Println("DeleteMessageBatchList error:", err)
		}
		fmt.Println("DeleteMessageBatchList output:", DeleteMessageBatchOutput)

	}
	return ctrl.Result{RequeueAfter: time.Second * r.RequeueAfter}, nil
}

func (r *SecretsRotationMappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awssecretsoperatorv1.SecretsRotationMapping{}).
		Complete(r)
}

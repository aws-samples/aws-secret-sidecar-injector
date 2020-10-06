package main

import (
	"encoding/json"
	"testing"

	"github.com/bmizerany/assert"
	jsonpatch "github.com/evanphx/json-patch"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// newRequest generating an AdmissionReview Resource
func newRequest(pod corev1.Pod) v1.AdmissionReview {
	object, _ := json.Marshal(pod)
	return v1.AdmissionReview{
		Request: &v1.AdmissionRequest{
			Name:      "DASA",
			Kind:      metav1.GroupVersionKind{Group: "", Kind: "Admission", Version: "v1"},
			Namespace: "DDS",
			UID:       "1234",
			// UserInfo:  nil,
			Resource: metav1.GroupVersionResource{Group: "", Resource: "pods", Version: "v1"},
			Object: runtime.RawExtension{
				Raw: object,
			},
		},
	}
}

// generateExamplePod resource
func generateExamplePod() corev1.Pod {
	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "new",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"secrets.k8s.aws/secret-arn":             "arn:aws:secretsmanager:us-east-1:123456789012:secret:database-password-hlRvvF",
				"secrets.k8s.aws/sidecarInjectorWebhook": "enabled",
			},
		},
	}
}

// execTest given a pod and an assert function to check the mutated result
func execTest(pod corev1.Pod, t *testing.T, assertFn func(outcome corev1.Pod)) {
	res := HandleNewReview(newRequest(pod))
	if !res.Allowed || res.Patch == nil {
		t.Error(res.Result.Message)
	}
	patchObj, err := jsonpatch.DecodePatch(res.Patch)
	if err != nil {
		t.Fatal(err)
	}
	podBytes, err := json.Marshal(pod)
	if err != nil {
		t.Fatal(err)
	}
	patchedPodBytes, err := patchObj.Apply(podBytes)
	if err != nil {
		t.Fatal(err)
	}
	patchedPod := corev1.Pod{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(patchedPodBytes, nil, &patchedPod); err != nil {
		t.Error(err)
	}
	assertFn(patchedPod)
}

//TestAccepting resources that do not have the webhook enabled
func TestAccepting(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"secrets.k8s.aws/secret-arn": "batata",
			},
		},
	}
	res := HandleNewReview(newRequest(pod))
	if !res.Allowed {
		t.Error("Blocking requests without secrets.k8s.aws/sidecarInjectorWebhook")
	}

	pod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"secrets.k8s.aws/secret-arn":             "batata",
				"secrets.k8s.aws/sidecarInjectorWebhook": "enabled",
			},
		},
	}
	res = HandleNewReview(newRequest(pod))
	if res.Result.Message != "Not a valid ARN: batata" || res.Allowed != false {
		t.Error("Accepting invalid ARNs")
	}
}

//TestBasic with one secret only
func TestBasic(t *testing.T) {
	pod := generateExamplePod()
	execTest(pod, t, func(out corev1.Pod) {
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[0].MountPath, "/secrets/database-password-hlRvvF", "wrong mount path")
	})
}

//TestMultipleSecrets thus multiple mounts
func TestMultipleSecrets(t *testing.T) {
	pod := generateExamplePod()
	pod.ObjectMeta.Annotations["secrets.k8s.aws/secret-arn"] = "arn:aws:secretsmanager:us-east-1:123456789012:secret:database-password-hlRvvF,arn:aws:secretsmanager:us-east-1:123456789012:secret:s3-bucker-2312"
	execTest(pod, t, func(out corev1.Pod) {
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[0].MountPath, "/secrets/database-password-hlRvvF", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[0].SubPath, "secrets/database-password-hlRvvF", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[1].MountPath, "/secrets/s3-bucker-2312", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[1].SubPath, "secrets/s3-bucker-2312", "wrong mount path")
	})
}

//TestInitContainer configuration
func TestInitContainer(t *testing.T) {
	pod := generateExamplePod()
	sidecarImage = "hub.docker/init-container"
	execTest(pod, t, func(out corev1.Pod) {
		assert.Equal(t, out.Spec.InitContainers[0].Name, initContainerName)
		assert.Equal(t, out.Spec.InitContainers[0].Image, sidecarImage)
		assert.Equal(t, out.Spec.InitContainers[0].VolumeMounts[0].MountPath, "/tmp")
		assert.Equal(t, out.Spec.InitContainers[0].VolumeMounts[0].Name, secretVolumeName)

		assert.Equal(t, out.Spec.Volumes[0].Name, secretVolumeName)
	})
}

//TestExistingVolumes where the empty-vol is added
func TestExistingVolumes(t *testing.T) {
	pod := generateExamplePod()
	existingVolumeName := "existing-volume"
	pod.Spec.Volumes = []corev1.Volume{
		{
			Name:         existingVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory}},
		},
	}
	sidecarImage = "hub.docker/init-container"
	execTest(pod, t, func(out corev1.Pod) {
		assert.Equal(t, out.Spec.Volumes[0].Name, existingVolumeName)
		assert.Equal(t, out.Spec.Volumes[1].Name, secretVolumeName)
	})
}

// VerifyTestMultipleSecretsMountAndContainers run a test with 3 secrets and 2 containers
func TestMultipleSecretsMountAndContainers(t *testing.T) {
	pod := generateExamplePod()
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name: "new-2",
	})
	pod.ObjectMeta.Annotations["secrets.k8s.aws/secret-arn"] = "arn:aws:secretsmanager:us-east-1:123456789012:secret:database-password-hlRvvF:/var/my-db-pass,arn:aws:secretsmanager:us-east-1:123456789012:secret:s3-bucker-2312,arn:aws:secretsmanager:us-east-1:123456789012:secret:s3-bucker-2312:/var/log/secreto"
	execTest(pod, t, func(out corev1.Pod) {
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[0].MountPath, "/var/my-db-pass", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[0].SubPath, "var/my-db-pass", "wrong mount path")

		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[1].MountPath, "/secrets/s3-bucker-2312", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[1].SubPath, "secrets/s3-bucker-2312", "wrong mount path")

		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[2].MountPath, "/var/log/secreto", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[0].VolumeMounts[2].SubPath, "var/log/secreto", "wrong mount path")

		assert.Equal(t, out.Spec.Containers[1].VolumeMounts[0].MountPath, "/var/my-db-pass", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[1].VolumeMounts[1].MountPath, "/secrets/s3-bucker-2312", "wrong mount path")
		assert.Equal(t, out.Spec.Containers[1].VolumeMounts[2].MountPath, "/var/log/secreto", "wrong mount path")
	})
}

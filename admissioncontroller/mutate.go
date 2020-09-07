package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

const (
	initContainerName   = "secrets-init-container"
	secretVolumeName    = "secret-vol"
	annotationSecretArn = "secrets.k8s.aws/secret-arn"
	annotationsWebHook  = "secrets.k8s.aws/sidecarInjectorWebhook"
)

func HandleNewReview(ar v1.AdmissionReview) *v1.AdmissionResponse {
	klog.V(2).Info("mutating pods")
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		klog.Errorf("expect resource to be %s", podResource)
		return nil
	}
	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		klog.Error(err)
		return toV1AdmissionResponse(err)
	}

	req := ar.Request
	klog.V(2).Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	if !shouldPatch(&pod) {

		klog.V(2).Infof("SecretInjector Name=%v (%v) UID=%v: skip", req.Name, pod.Name, req.UID)
		reviewResponse := v1.AdmissionResponse{
			Allowed: true,
		}
		klog.Info(&reviewResponse)
		return &reviewResponse
	}
	var patchBytes []byte
	patchOperations, err := createPatch(&pod, sidecarImage)
	if err != nil {
		return toV1AdmissionResponse(err)
	}
	patchBytes, err = json.Marshal(patchOperations)
	if err != nil {
		return toV1AdmissionResponse(err)
	}

	return &v1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1.PatchType {
			pt := v1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func shouldPatch(pod *corev1.Pod) bool {
	inject_status, _ := pod.ObjectMeta.Annotations[annotationsWebHook]
	if inject_status != "enabled" {
		return false
	}
	_, arn_ok := pod.ObjectMeta.Annotations[annotationSecretArn]
	if arn_ok == false {
		return false
	}
	return !hasContainer(pod.Spec.InitContainers, initContainerName)
}

func hasContainer(containers []corev1.Container, containerName string) bool {
	for _, container := range containers {
		if container.Name == containerName {
			return true
		}
	}
	return false
}

// create mutation patch for resoures
func createPatch(pod *corev1.Pod, initContainerImage string) (patch []patchOperation, err error) {

	// addInitContainer
	initContainer := corev1.Container{
		Image: initContainerImage,
		Name:  initContainerName,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      secretVolumeName,
				MountPath: "/tmp"},
		},
		Env: []corev1.EnvVar{
			{
				Name: "SECRET_ARN",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.annotations['secrets.k8s.aws/secret-arn']",
					},
				},
			},
		},
	}
	patch = append(patch, addContainer(pod.Spec.InitContainers, []corev1.Container{initContainer}, "/spec/initContainers")...)

	//Add Volume
	secretVolume := corev1.Volume{
		Name: secretVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
	patch = append(patch, addVolume(pod.Spec.Volumes, []corev1.Volume{secretVolume}, "/spec/volumes")...)

	//add Volume Mounts
	annotation := pod.ObjectMeta.Annotations[annotationSecretArn]
	arns := strings.Split(annotation, ",")
	volumeMounts := []corev1.VolumeMount{}
	for _, arn := range arns {
		_, mountPath, err := GetArnAndMountPath(arn)
		if err != nil {
			return patch, err
		}
		if !filepath.IsAbs(mountPath) {
			fmt.Errorf("Mount path must be absolute: %s", mountPath)
		}

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      secretVolumeName,
			MountPath: mountPath,
			ReadOnly:  true,
			SubPath:   mountPath[1:], //subpath is relative to empty dir
		})
	}
	patch = append(patch, addVolumeMount(pod.Spec.Containers, volumeMounts)...)
	return patch, nil
	// return json.Marshal(patch)
}

//adds volume mounts to each container
func addVolumeMount(containers []corev1.Container, mounts []corev1.VolumeMount) (patch []patchOperation) {
	for id, container := range containers {
		first := len(container.VolumeMounts) == 0
		for _, mount := range mounts {
			path := "/spec/containers/" + strconv.Itoa(id) + "/volumeMounts"
			var value interface{}
			value = mount
			if first {
				first = false
				value = []corev1.VolumeMount{mount}
			} else {
				path = path + "/-"
			}
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  path,
				Value: value,
			})
		}
	}
	return patch
}

func addContainer(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}

		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

//arn:aws:secretsmanager:us-east-1:123456789012:secret:database-password-hlRvvF:/var/log/secret
func GetArnAndMountPath(secretStr string) (secretArn string, mountPath string, err error) {
	// regex parse to extract arn and mount path
	var re = regexp.MustCompile(`(?m)(.*:secret:[^:]*):?(.*)`)
	match := re.FindStringSubmatch(secretStr)
	if len(match) == 3 {
		//<secret-arn>:<mounting-path>
		secretArn = match[1]
		mountPath = match[2]
	} else {
		secretArn = secretStr
	}

	if !arn.IsARN(secretArn) {
		return "", "", fmt.Errorf("Not a valid ARN: %s", secretArn)
	}
	arnobj, err := arn.Parse(secretArn)
	if err != nil {
		return "", "", fmt.Errorf("Can not parse arn: %s %v \n", secretArn, err)
	}
	if mountPath == "" {
		secretName := strings.Split(arnobj.Resource, "secret:")[1]
		mountPath = filepath.Join("/secrets", secretName)
	}
	return secretArn, mountPath, nil
}

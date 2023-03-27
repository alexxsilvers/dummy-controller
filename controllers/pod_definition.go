package controllers

import (
	"fmt"
	"os"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"

	dummyv1alpha1 "github.com/alexxsilvers/k8s-dummy-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *DummyReconciler) createPodDefinition(dummy *dummyv1alpha1.Dummy) (*corev1.Pod, error) {
	podImage, err := imageForPod()
	if err != nil {
		return nil, fmt.Errorf("get image for pod %w", err)
	}

	labels := labelsForPod(dummy.Name, podImage)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", dummy.Name, "pod"),
			Namespace: dummy.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            podImage.name,
					Image:           podImage.fullImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
		},
	}

	// set ownerRef for this pod. If dummy will be deleted, this pod will delete automatically
	err = ctrl.SetControllerReference(dummy, pod, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("set ownerRef for pod")
	}

	return pod, nil
}

type image struct {
	name      string
	tag       string
	fullImage string
}

// imageForPod gets the Operand image which is managed by this controller
// from the POD_IMAGE environment variable defined in the config/manager/manager.yaml
func imageForPod() (image, error) {
	var imageEnvVar = "POD_IMAGE"
	imageStr, found := os.LookupEnv(imageEnvVar)
	if !found {
		return image{}, fmt.Errorf("unable to find %s environment variable with the image", imageEnvVar)
	}

	imageSplited := strings.Split(imageStr, ":")
	if len(imageSplited) != 2 {
		return image{}, fmt.Errorf("invalid pod image value")
	}

	return image{
		name:      imageSplited[0],
		tag:       imageSplited[1],
		fullImage: imageStr,
	}, nil
}

// labelsForPod returns the labels for selecting the resources
func labelsForPod(name string, image image) map[string]string {
	return map[string]string{"app.kubernetes.io/name": "Dummy",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/version":    image.tag,
		"app.kubernetes.io/part-of":    "dummy-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

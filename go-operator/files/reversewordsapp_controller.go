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
	"github.com/go-logr/logr"
	appsv1alpha1 "github.com/GHUSER/reverse-words-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReverseWordsAppReconciler reconciles a ReverseWordsApp object
type ReverseWordsAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Finalizer for our objects
const reverseWordsAppFinalizer = "finalizer.reversewordsapp.apps.linuxera.org"

// +kubebuilder:rbac:groups=apps.linuxera.org,resources=reversewordsapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.linuxera.org,resources=reversewordsapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.linuxera.org,resources=reversewordsapps/finalizers,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *ReverseWordsAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("reversewordsapp", req.NamespacedName)

	// Fetch the ReverseWordsApp instance
	instance := &appsv1alpha1.ReverseWordsApp{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("ReverseWordsApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ReverseWordsApp")
		return ctrl.Result{}, err
	}

	// Check if the CR is marked to be deleted
	isInstanceMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isInstanceMarkedToBeDeleted {
		log.Info("Instance marked for deletion, running finalizers")
		if contains(instance.GetFinalizers(), reverseWordsAppFinalizer) {
			// Run the finalizer logic
			err := r.finalizeReverseWordsApp(log, instance)
			if err != nil {
				// Don't remove the finalizer if we failed to finalize the object
				return ctrl.Result{}, err
			}
			log.Info("Instance finalizers completed")
			// Remove finalizer once the finalizer logic has run
			controllerutil.RemoveFinalizer(instance, reverseWordsAppFinalizer)
			err = r.Update(ctx, instance)
			if err != nil {
				// If the object update fails, requeue
				return ctrl.Result{}, err
			}
		}
		log.Info("Instance can be deleted now")
		return ctrl.Result{}, nil
	}

	// Add Finalizers to the CR
	if !contains(instance.GetFinalizers(), reverseWordsAppFinalizer) {
		if err := r.addFinalizer(log, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Deployment object
	result, err := r.reconcileDeployment(instance, log)
	if err != nil {
		return result, err
	}
	// Reconcile Service object
	result, err = r.reconcileService(instance, log)
	if err != nil {
		return result, err
	}

	// The CR status is updated in the Deployment reconcile method

	return ctrl.Result{}, nil
}

func (r *ReverseWordsAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.ReverseWordsApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *ReverseWordsAppReconciler) reconcileDeployment(cr *appsv1alpha1.ReverseWordsApp, log logr.Logger) (ctrl.Result, error) {
	// Define a new Deployment object
	deployment := newDeploymentForCR(cr)

	// Set ReverseWordsApp instance as the owner and controller of the Deployment
	if err := ctrl.SetControllerReference(cr, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if this Deployment already exists
	deploymentFound := &appsv1.Deployment{}
	err := r.Get(context.Background(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deploymentFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Create(context.Background(), deployment)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Requeue the object to update its status
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		// Deployment already exists
		log.Info("Deployment already exists", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
	}

	// Ensure deployment replicas match the desired state
	if !reflect.DeepEqual(deploymentFound.Spec.Replicas, deployment.Spec.Replicas) {
		log.Info("Current deployment replicas do not match ReverseWordsApp configured Replicas")
		// Update the replicas
		err = r.Update(context.Background(), deployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
			return ctrl.Result{}, err
		}
	}
	// Ensure deployment container image match the desired state, returns true if deployment needs to be updated
	if checkDeploymentImage(deploymentFound, deployment) {
		log.Info("Current deployment image version do not match ReverseWordsApp configured version")
		// Update the image
		err = r.Update(context.Background(), deployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
			return ctrl.Result{}, err
		}
	}

	// Check if the deployment is ready
	deploymentReady := isDeploymentReady(deploymentFound)

	// Create list options for listing deployment pods
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(deploymentFound.Namespace),
		client.MatchingLabels(deploymentFound.Labels),
	}
	// List the pods for this ReverseWordsApp deployment
	err = r.List(context.Background(), podList, listOpts...)
	if err != nil {
		log.Error(err, "Failed to list Pods.", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
		return ctrl.Result{}, err
	}
	// Get running Pods from listing above (if any)
	podNames := getRunningPodNames(podList.Items)
	if deploymentReady {
		// Update the status to ready
		cr.Status.AppPods = podNames
		cr.SetCondition(appsv1alpha1.ConditionTypeReverseWordsDeploymentNotReady, false)
		cr.SetCondition(appsv1alpha1.ConditionTypeReady, true)
	} else {
		// Update the status to not ready
		cr.Status.AppPods = podNames
		cr.SetCondition(appsv1alpha1.ConditionTypeReverseWordsDeploymentNotReady, true)
		cr.SetCondition(appsv1alpha1.ConditionTypeReady, false)
	}
	// Reconcile the new status for the instance
	cr, err = r.updateReverseWordsAppStatus(cr, log)
	if err != nil {
		log.Error(err, "Failed to update ReverseWordsApp Status.")
		return ctrl.Result{}, err
	}
	// Deployment reconcile finished
	return ctrl.Result{}, nil
}

// updateReverseWordsAppStatus updates the Status of a given CR
func (r *ReverseWordsAppReconciler) updateReverseWordsAppStatus(cr *appsv1alpha1.ReverseWordsApp, log logr.Logger) (*appsv1alpha1.ReverseWordsApp, error) {
	reverseWordsApp := &appsv1alpha1.ReverseWordsApp{}
	err := r.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, reverseWordsApp)
	if err != nil {
		return reverseWordsApp, err
	}

	if !reflect.DeepEqual(cr.Status, reverseWordsApp.Status) {
		log.Info("Updating ReverseWordsApp Status.")
		// We need to update the status
		err = r.Status().Update(context.Background(), cr)
		if err != nil {
			return cr, err
		}
		updatedReverseWordsApp := &appsv1alpha1.ReverseWordsApp{}
		err = r.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, updatedReverseWordsApp)
		if err != nil {
			return cr, err
		}
		cr = updatedReverseWordsApp.DeepCopy()
	}
	return cr, nil

}

// addFinalizer adds a given finalizer to a given CR
func (r *ReverseWordsAppReconciler) addFinalizer(log logr.Logger, cr *appsv1alpha1.ReverseWordsApp) error {
	log.Info("Adding Finalizer for the ReverseWordsApp")
	controllerutil.AddFinalizer(cr, reverseWordsAppFinalizer)

	// Update CR
	err := r.Update(context.Background(), cr)
	if err != nil {
		log.Error(err, "Failed to update ReverseWordsApp with finalizer")
		return err
	}
	return nil
}

// finalizeReverseWordsApp runs required tasks before deleting the objects owned by the CR
func (r *ReverseWordsAppReconciler) finalizeReverseWordsApp(log logr.Logger, cr *appsv1alpha1.ReverseWordsApp) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	log.Info("Successfully finalized ReverseWordsApp")
	return nil
}

func (r *ReverseWordsAppReconciler) reconcileService(cr *appsv1alpha1.ReverseWordsApp, log logr.Logger) (ctrl.Result, error) {
	// Define a new Service object
	service := newServiceForCR(cr)

	// Set ReverseWordsApp instance as the owner and controller of the Service
	if err := controllerutil.SetControllerReference(cr, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if this Service already exists
	serviceFound := &corev1.Service{}
	err := r.Get(context.Background(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, serviceFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Create(context.Background(), service)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Service created successfully - don't requeue
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		// Service already exists
		log.Info("Service already exists", "Service.Namespace", serviceFound.Namespace, "Service.Name", serviceFound.Name)
	}
	// Service reconcile finished
	return ctrl.Result{}, nil
}

// Returns a new deployment without replicas configured
// replicas will be configured in the sync loop
func newDeploymentForCR(cr *appsv1alpha1.ReverseWordsApp) *appsv1.Deployment {
	labels := map[string]string{
		"app": cr.Name,
	}
	replicas := cr.Spec.Replicas
	// Minimum replicas will be 1
	if replicas == 0 {
		replicas = 1
	}
	appVersion := "latest"
	if cr.Spec.AppVersion != "" {
		appVersion = cr.Spec.AppVersion
	}
	// TODO:Check if application version exists
	containerImage := "quay.io/mavazque/reversewords:" + appVersion
	probe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.FromInt(8080),
			},
		},
		InitialDelaySeconds: 5,
		TimeoutSeconds:      2,
		PeriodSeconds:       15,
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dp-" + cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: containerImage,
							Name:  "reversewords",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:          "reversewords",
								},
							},
							LivenessProbe:  probe,
							ReadinessProbe: probe,
						},
					},
				},
			},
		},
	}
}

// Returns a new service
func newServiceForCR(cr *appsv1alpha1.ReverseWordsApp) *corev1.Service {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-" + cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 8080,
				},
			},
		},
	}
}

// isDeploymentReady returns a true bool if the deployment has all its pods ready
func isDeploymentReady(deployment *appsv1.Deployment) bool {
	configuredReplicas := deployment.Status.Replicas
	readyReplicas := deployment.Status.ReadyReplicas
	deploymentReady := false
	if configuredReplicas == readyReplicas {
		deploymentReady = true
	}
	return deploymentReady
}

// getRunningPodNames returns the pod names for the pods running in the array of pods passed in
func getRunningPodNames(pods []corev1.Pod) []string {
	// Create an empty []string, so if no podNames are returned, instead of nil we get an empty slice
	var podNames []string = make([]string, 0)
	for _, pod := range pods {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			podNames = append(podNames, pod.Name)
		}
	}
	return podNames
}

// checkDeploymentImage returns wether the deployment image is different or not
func checkDeploymentImage(current *appsv1.Deployment, desired *appsv1.Deployment) bool {
	for _, curr := range current.Spec.Template.Spec.Containers {
		for _, des := range desired.Spec.Template.Spec.Containers {
			// Only compare the images of containers with the same name
			if curr.Name == des.Name {
				if curr.Image != des.Image {
					return true
				}
			}
		}
	}
	return false
}

// contains returns true if a string is found on a slice
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

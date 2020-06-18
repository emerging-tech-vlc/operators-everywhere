package reversewordsapp

import (
    "context"
    "reflect"
    emergingtechv1alpha1 "github.com/GHUSER/reverse-words-operator/pkg/apis/emergingtech/v1alpha1"
    "github.com/go-logr/logr"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/apimachinery/pkg/util/intstr"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/handler"
    "sigs.k8s.io/controller-runtime/pkg/manager"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
    logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
    "sigs.k8s.io/controller-runtime/pkg/source"
    appsv1 "k8s.io/api/apps/v1"
)

var log = logf.Log.WithName("controller_reversewordsapp")

// Add creates a new ReverseWordsApp Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
    return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
    return &ReconcileReverseWordsApp{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
    // Create a new controller
    c, err := controller.New("reversewordsapp-controller", mgr, controller.Options{Reconciler: r})
    if err != nil {
        return err
    }

    // Watch for changes to primary resource ReverseWordsApp
    err = c.Watch(&source.Kind{Type: &emergingtechv1alpha1.ReverseWordsApp{}}, &handler.EnqueueRequestForObject{})
    if err != nil {
        return err
    }

    // Watch for changes to secondary resources Deployments and Services and requeue the owner ReverseWordsApp
    ownedObjects := []runtime.Object{
        &appsv1.Deployment{},
        &corev1.Service{},
    }

    for _, ownedObject := range ownedObjects {
        err = c.Watch(&source.Kind{Type: ownedObject}, &handler.EnqueueRequestForOwner{
            IsController: true,
            OwnerType:    &emergingtechv1alpha1.ReverseWordsApp{},
        })
        if err != nil {
            return err
        }
    }

    return nil
}
// blank assignment to ensure that ReconcileReverseWordsApp implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileReverseWordsApp{}

// ReconcileReverseWordsApp reconciles a ReverseWordsApp object
type ReconcileReverseWordsApp struct {
    // This client, initialized using mgr.Client() above, is a split client
    // that reads objects from the cache and writes to the apiserver
    client client.Client
    scheme *runtime.Scheme
}

// Finalizer for our objects
const reverseWordsAppFinalizer = "finalizer.reversewordsapp.emergingtech.vlc"

// Reconcile reads that state of the cluster for a ReverseWordsApp object and makes changes based on the state read
// and what is in the ReverseWordsApp.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileReverseWordsApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
    reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
    reqLogger.Info("Reconciling ReverseWordsApp")

    // Fetch the ReverseWordsApp instance
    instance := &emergingtechv1alpha1.ReverseWordsApp{}
    err := r.client.Get(context.TODO(), request.NamespacedName, instance)
    if err != nil {
        if errors.IsNotFound(err) {
            // Request object not found, could have been deleted after reconcile request.
            // Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
            // Return and don't requeue
            return reconcile.Result{}, nil
        }
        // Error reading the object - requeue the request.
        return reconcile.Result{}, err
    }

    // Check if the CR is marked to be deleted
    isInstanceMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
    if isInstanceMarkedToBeDeleted {
        reqLogger.Info("Instance marked for deletion, running finalizers")
        if contains(instance.GetFinalizers(), reverseWordsAppFinalizer) {
            // Run the finalizer logic
            err := r.finalizeReverseWordsApp(reqLogger, instance)
            if err != nil {
                // Don't remove the finalizer if we failed to finalize the object
                return reconcile.Result{}, err
            }
            reqLogger.Info("Instance finalizers completed")
            // Remove finalizer once the finalizer logic has run
            controllerutil.RemoveFinalizer(instance, reverseWordsAppFinalizer)
            err = r.client.Update(context.TODO(), instance)
            if err != nil {
                // If the object update fails, requeue
				return reconcile.Result{}, err
            }
        }
        reqLogger.Info("Instance can be deleted now")
        return reconcile.Result{}, nil
    }

    // Add Finalizers to the CR
    if !contains(instance.GetFinalizers(), reverseWordsAppFinalizer) {
        if err := r.addFinalizer(reqLogger, instance); err != nil {
            return reconcile.Result{}, err
		}
    }

    // Reconcile Deployment object
    result, err := r.reconcileDeployment(instance, reqLogger)
    if err != nil {
        return result, err
    }
    // Reconcile Service object
    result, err = r.reconcileService(instance, reqLogger)
    if err != nil {
        return result, err
    }

    // The CR status is updated in the Deployment reconcile method

    return reconcile.Result{}, err
}

func (r *ReconcileReverseWordsApp) reconcileDeployment(cr *emergingtechv1alpha1.ReverseWordsApp, reqLogger logr.Logger) (reconcile.Result, error) {
    // Define a new Deployment object
    deployment := newDeploymentForCR(cr)

    // Set ReverseWordsApp instance as the owner and controller of the Deployment
    if err := controllerutil.SetControllerReference(cr, deployment, r.scheme); err != nil {
        return reconcile.Result{}, err
    }

    // Check if this Deployment already exists
    deploymentFound := &appsv1.Deployment{}
    err := r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deploymentFound)
    if err != nil && errors.IsNotFound(err) {
        reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
        err = r.client.Create(context.TODO(), deployment)
        if err != nil {
            return reconcile.Result{}, err
        }
        // Get existing deployment again
        //deploymentFound = &appsv1.Deployment{}
        //err = r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deploymentFound)
        // Requeue the object to update its status
        return reconcile.Result{Requeue: true}, nil
    } else if err != nil {
        return reconcile.Result{}, err
    } else {
        // Deployment already exists
        reqLogger.Info("Deployment already exists", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
    }

    // Ensure deployment replicas match the desired state
    if !reflect.DeepEqual(deploymentFound.Spec.Replicas, deployment.Spec.Replicas) {
        reqLogger.Info("Current deployment replicas do not match ReverseWordsApp configured Replicas")
        // Update the replicas
        err = r.client.Update(context.TODO(), deployment)
        if err != nil {
            reqLogger.Error(err, "Failed to update Deployment.", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
            return reconcile.Result{}, err
        }
    }
    // Ensure deployment container image match the desired state, returns true if deployment needs to be updated
    if checkDeploymentImage(deploymentFound, deployment) {
        reqLogger.Info("Current deployment image version do not match ReverseWordsApp configured version")
        // Update the image
        err = r.client.Update(context.TODO(), deployment)
        if err != nil {
            reqLogger.Error(err, "Failed to update Deployment.", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
            return reconcile.Result{}, err
        }
    }

    // Check if the deployment is ready

    deploymentReady := isDeploymentReady(deploymentFound) 
    
    if deploymentReady {
        // List the pods for this ReverseWordsApp deployment
        podList := &corev1.PodList{}
        listOpts := []client.ListOption{
            client.InNamespace(deploymentFound.Namespace),
            client.MatchingLabels(deploymentFound.Labels),
        }
        err = r.client.List(context.TODO(), podList, listOpts...)
        if err != nil {
            reqLogger.Error(err, "Failed to list Pods.", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
            return reconcile.Result{}, err
        }

        podNames := getRunningPodNames(podList.Items)
        // Update the status
        cr.Status.AppPods = podNames
        cr.SetCondition(emergingtechv1alpha1.ConditionTypeReverseWordsDeploymentNotReady, false)    
        cr.SetCondition(emergingtechv1alpha1.ConditionTypeReady, true)
    } else {
        cr.SetCondition(emergingtechv1alpha1.ConditionTypeReverseWordsDeploymentNotReady, true)    
        cr.SetCondition(emergingtechv1alpha1.ConditionTypeReady, false)
    }

    // Reconcile the new status for the instance
    cr, err = r.updateReverseWordsAppStatus(cr, reqLogger)
    if err != nil {
        reqLogger.Error(err, "Failed to update ReverseWordsApp Status.")
		return reconcile.Result{}, err
    }

    // Deployment reconcile finished
    return reconcile.Result{}, nil
}

// updateReverseWordsAppStatus updates the Status of a given CR
func (r *ReconcileReverseWordsApp) updateReverseWordsAppStatus(cr *emergingtechv1alpha1.ReverseWordsApp, reqLogger logr.Logger) (*emergingtechv1alpha1.ReverseWordsApp, error) {
    reverseWordsApp := &emergingtechv1alpha1.ReverseWordsApp{}
    err := r.client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, reverseWordsApp)
    if err != nil {
		return reverseWordsApp, err
    }

    if !reflect.DeepEqual(cr.Status, reverseWordsApp.Status) {
        reqLogger.Info("Updating ReverseWordsApp Status.")
        // We need to update the status      
        err = r.client.Status().Update(context.TODO(), cr)
        if err != nil {
			return cr, err
        }
        updatedReverseWordsApp := &emergingtechv1alpha1.ReverseWordsApp{}
        err = r.client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, updatedReverseWordsApp)
        if err != nil {
			return cr, err
        }
        cr = updatedReverseWordsApp.DeepCopy()
    }
    return cr, nil

}

// addFinalizer adds a given finalizer to a given CR
func (r *ReconcileReverseWordsApp) addFinalizer(reqLogger logr.Logger, cr *emergingtechv1alpha1.ReverseWordsApp) error {
	reqLogger.Info("Adding Finalizer for the ReverseWordsApp")
	controllerutil.AddFinalizer(cr, reverseWordsAppFinalizer)

	// Update CR
	err := r.client.Update(context.TODO(), cr)
	if err != nil {
		reqLogger.Error(err, "Failed to update ReverseWordsApp with finalizer")
		return err
	}
	return nil
}

// finalizeReverseWordsApp runs required tasks before deleting the objects owned by the CR
func (r *ReconcileReverseWordsApp) finalizeReverseWordsApp(reqLogger logr.Logger, cr *emergingtechv1alpha1.ReverseWordsApp) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	reqLogger.Info("Successfully finalized ReverseWordsApp")
	return nil
}

func (r *ReconcileReverseWordsApp) reconcileService(cr *emergingtechv1alpha1.ReverseWordsApp, reqLogger logr.Logger) (reconcile.Result, error) {
    // Define a new Service object
    service := newServiceForCR(cr)

    // Set ReverseWordsApp instance as the owner and controller of the Service
    if err := controllerutil.SetControllerReference(cr, service, r.scheme); err != nil {
        return reconcile.Result{}, err
    }

    // Check if this Service already exists
    serviceFound := &corev1.Service{}
    err := r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, serviceFound)
    if err != nil && errors.IsNotFound(err) {
        reqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
        err = r.client.Create(context.TODO(), service)
        if err != nil {
            return reconcile.Result{}, err
        }
        // Service created successfully - don't requeue
        return reconcile.Result{}, nil
    } else if err != nil {
        return reconcile.Result{}, err
    } else {
        // Service already exists
        reqLogger.Info("Service already exists", "Service.Namespace", serviceFound.Namespace, "Service.Name", serviceFound.Name)
    }
    // Service reconcile finished
    return reconcile.Result{}, nil
}


// Returns a new deployment without replicas configured
// replicas will be configured in the sync loop
func newDeploymentForCR(cr *emergingtechv1alpha1.ReverseWordsApp) *appsv1.Deployment {
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
        TimeoutSeconds: 2,
        PeriodSeconds: 15,
    }
    return &appsv1.Deployment{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "apps/v1",
            Kind:       "Deployment",
        },
        ObjectMeta: metav1.ObjectMeta{
            Name:      "deployment-" + cr.Name,
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
                                    Name: "reversewords",
                                },
                            },
                            LivenessProbe: probe,
                            ReadinessProbe: probe,
                        },
                    },
                },
            },
        },
    }
}

// Returns a new service
func newServiceForCR(cr *emergingtechv1alpha1.ReverseWordsApp) *corev1.Service {
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
            Labels: labels,
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

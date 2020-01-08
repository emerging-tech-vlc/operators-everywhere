package reversewordsapp

import (
    "context"
    "reflect"
    emergingtechv1alpha1 "github.com/mvazquezc/reverse-words-operator/pkg/apis/emergingtech/v1alpha1"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
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

    // Watch for changes to secondary resource Deployments and requeue the owner ReverseWordsApp
    err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
        IsController: true,
        OwnerType:    &emergingtechv1alpha1.ReverseWordsApp{},
    })
    if err != nil {
        return err
    }

    // Watch for changes to secondary resource Services and requeue the owner ReverseWordsApp
    err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
        IsController: true,
        OwnerType:    &emergingtechv1alpha1.ReverseWordsApp{},
    })
    if err != nil {
        return err
    }

    return nil
}

var _ reconcile.Reconciler = &ReconcileReverseWordsApp{}

// ReconcileReverseWordsApp reconciles a ReverseWordsApp object
type ReconcileReverseWordsApp struct {
    // This client, initialized using mgr.Client() above, is a split client
    // that reads objects from the cache and writes to the apiserver
    client client.Client
    scheme *runtime.Scheme
}

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

    // Get a deployment for our application
    // Define a new Deployment object
    deployment := newDeploymentForCR(instance)

    // Get a service for our application
    // Define a new Service object
    service := newServiceForCR(instance)

    // Set ReverseWordsApp instance as the owner and controller of the Deployment
    if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
        return reconcile.Result{}, err
    }
    // Set ReverseWordsApp instance as the owner and controller of the Service
    if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
        return reconcile.Result{}, err
    }

    // Get configured replicas and release from the Spec
    specReplicas := instance.Spec.Replicas

    // Check if this Deployment already exists
    deploymentFound := &appsv1.Deployment{}
    err = r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deploymentFound)
    if err != nil && errors.IsNotFound(err) {
        reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
        err = r.client.Create(context.TODO(), deployment)
        if err != nil {
            return reconcile.Result{}, err
        }
        // Deployment created successfully - don't requeue
        return reconcile.Result{}, nil
    } else if err != nil {
        return reconcile.Result{}, err
    } else {
        // Deployment already exists
        reqLogger.Info("Deployment already exists", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
    }

    // Check if this Service already exists
    serviceFound := &corev1.Service{}
    err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, serviceFound)
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

    // Ensure deployment replicas match the desired state
    if *deploymentFound.Spec.Replicas != specReplicas {
        log.Info("Current deployment replicas do not match ReverseWordsApp configured Replicas")
        deploymentFound.Spec.Replicas = &specReplicas
        // Update the replicas
        err = r.client.Update(context.TODO(), deploymentFound)
        if err != nil {
            reqLogger.Error(err, "Failed to update Deployment.", "Deployment.Namespace", deploymentFound.Namespace, "Deployment.Name", deploymentFound.Name)
            return reconcile.Result{}, err
        }
        // Spec updated - return and requeue (so we can update status)
        return reconcile.Result{Requeue: true}, nil
    }

    // Update the ReverseWordsApp status with the pod names
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

    // Update the appPods if needed
    if !reflect.DeepEqual(podNames, instance.Status.AppPods) {
        instance.Status.AppPods = podNames
        err := r.client.Status().Update(context.TODO(), instance)
        if err != nil {
            reqLogger.Error(err, "Failed to update ReverseWordsApp status.")
            return reconcile.Result{}, err
        }
        log.Info("Status updated")
    } else {
        log.Info("Status has not changed")
    }

    return reconcile.Result{}, nil
}

// Returns a new deployment without replicas configured
// replicas will be configured in the sync loop
func newDeploymentForCR(cr *emergingtechv1alpha1.ReverseWordsApp) *appsv1.Deployment {
    labels := map[string]string{
        "app": cr.Name,
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
            Selector: &metav1.LabelSelector{
                MatchLabels: labels,
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: labels,
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Image: "quay.io/mavazque/reversewords:latest",
                        Name:  "reversewords",
                        Ports: []corev1.ContainerPort{{
                            ContainerPort: 8080,
                            Name: "reversewords",
                        }},
                    }},
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

// getRunningPodNames returns the pod names for the pods running in the array of pods passed in
func getRunningPodNames(pods []corev1.Pod) []string {
    var podNames []string
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

/*
Copyright 2021.

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
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	mycontrollerv1 "github.com/mycontroller-org/mycontroller-operator/api/v1"
	mycCmap "github.com/mycontroller-org/server/v2/pkg/model/cmap"
	mycConfig "github.com/mycontroller-org/server/v2/pkg/model/config"
	mycServicefilter "github.com/mycontroller-org/server/v2/pkg/model/service_filter"
)

// MyControllerReconciler reconciles a MyController object
type MyControllerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mycontroller.org,resources=mycontrollers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mycontroller.org,resources=mycontrollers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mycontroller.org,resources=mycontrollers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyController object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *MyControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	log.Info("reconcile triggered", "namespace", req.Namespace, "name", req.Name)

	// fetch the MyController instance
	myController := &mycontrollerv1.MyController{}
	err := r.Get(ctx, req.NamespacedName, myController)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("MyController resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "failed to get MyController")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: myController.Name, Namespace: myController.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		err = r.removeMyController(ctx, myController)
		if err != nil {
			log.Error(err, "failed to remove existing resources")
			return ctrl.Result{}, err
		}
		err = r.setupMyController(ctx, myController)
		if err != nil {
			log.Error(err, "failed to setup mycontroller")
			return ctrl.Result{}, err
		}

		myController.Status.Phase = mycontrollerv1.MyControllerPhaseRunning
		if err = r.Client.Status().Update(ctx, myController); err != nil {
			log.Error(err, "failed to update status")
		}
		// deployed successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "failed to get Deployment")
		return ctrl.Result{}, err
	}

	// TODO: verify the existing installation status and reconcile, if there is a need

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log := mgr.GetLogger()

	// update platform
	err := updatePlatform(mgr)
	if err != nil {
		return err
	}

	// Adding the routev1
	if isOpenshift() {
		if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
			log.Error(err, "")
			return err
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&mycontrollerv1.MyController{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *MyControllerReconciler) removeMyController(ctx context.Context, m *mycontrollerv1.MyController) error {
	// delete existing resources and create new resources
	// delete deployment
	err := r.Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: m.Namespace, Name: m.Name}})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	// delete configmap
	err = r.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: m.Namespace, Name: m.Name}})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	// delete route
	if isOpenshift() {
		err = r.Delete(ctx, &routev1.Route{ObjectMeta: metav1.ObjectMeta{Namespace: m.Namespace, Name: m.Name}})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	// delete service
	err = r.Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: m.Namespace, Name: m.Name}})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *MyControllerReconciler) setupMyController(ctx context.Context, m *mycontrollerv1.MyController) error {
	// create configmap
	err := r.createConfigMap(ctx, m)
	if err != nil {
		return err
	}

	// deploy MyController
	err = r.deployMyController(ctx, m)
	if err != nil {
		return err
	}

	// create services
	err = r.createService(ctx, m)
	if err != nil {
		return err
	}

	// create route, only for openshift
	err = r.createRoute(ctx, m)
	if err != nil {
		return err
	}

	return nil
}

func (r *MyControllerReconciler) deployMyController(ctx context.Context, m *mycontrollerv1.MyController) error {
	log := ctrllog.FromContext(ctx)

	ls := labelsForMyController(m.Name)
	replicas := int32(1)

	// storage volumes
	dataVolume := corev1.Volume{
		Name:         "myc-data-dir",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	metricVolume := corev1.Volume{
		Name:         "myc-metric-dir",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}

	// update storage volumes
	storage := m.Spec.Storage
	if storage.SizeData != nil {
		err := r.updatePVC(ctx, m.Namespace, storage.StorageClassName, fmt.Sprintf("data-%s", m.Name), storage.SizeData, &dataVolume)
		if err != nil {
			return err
		}
	}
	if storage.SizeMetric != nil {
		err := r.updatePVC(ctx, m.Namespace, storage.StorageClassName, fmt.Sprintf("metric-%s", m.Name), storage.SizeMetric, &metricVolume)
		if err != nil {
			return err
		}
	}

	mycDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "influxdb",
							Image: "influxdb:1.8.4",
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8086,
								Name:          "influxdb",
							}},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "myc-metric-dir",
								MountPath: "/var/lib/influxdb",
							}},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 7000,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "health",
										Port:   intstr.FromInt(8086),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
						},
						{
							Image:   "quay.io/mycontroller/server:master",
							Name:    "mycontroller",
							Command: []string{"/app/mycontroller-server", "-config", "/app/mycontroller.yaml"},
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8080,
								Name:          "mycontroller",
							}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "myc-data-dir",
									MountPath: "/mc_home",
								},
								{
									Name:      "myc-config",
									MountPath: "/app/mycontroller.yaml",
									SubPath:   "mycontroller.yaml",
									ReadOnly:  true,
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 5000,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "api/status",
										Port:   intstr.FromInt(8080),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
						}},
					Volumes: []corev1.Volume{
						dataVolume,
						metricVolume,
						{
							Name: "myc-config",
							VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: m.Name},
							}},
						},
					},
				},
			},
		},
	}
	// Set MyController instance as the owner and controller
	ctrl.SetControllerReference(m, mycDeployment, r.Scheme)
	err := r.Create(ctx, mycDeployment)
	if err != nil {
		log.Error(err, "failed to create new Deployment", "Deployment.Namespace", mycDeployment.Namespace, "Deployment.Name", mycDeployment.Name)
		return err
	}

	return nil
}

func (r *MyControllerReconciler) updatePVC(ctx context.Context, namespace, storageClassName, claimName string, pvSize *resource.Quantity, volume *corev1.Volume) error {
	log := ctrllog.FromContext(ctx)
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: claimName, Namespace: namespace}, pvc)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create pvc
		pvc.ObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: claimName}
		storageClassNameTmp := storageClassName
		pvc.Spec = corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClassNameTmp,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"storage": *pvSize},
			},
		}
		err = r.Create(ctx, pvc)
		if err != nil {
			log.Error(err, "failed to create new pvc", "Namespace", namespace, "Name", claimName, "StorageClassName", storageClassName)
			return err
		}
	}
	// update into given volume
	volume.VolumeSource = corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
		ReadOnly:  false,
		ClaimName: claimName,
	}}
	return nil
}

func (r *MyControllerReconciler) createConfigMap(ctx context.Context, m *mycontrollerv1.MyController) error {
	log := ctrllog.FromContext(ctx)
	mycCfg := mycConfig.Config{
		Secret:    m.Spec.Secret,
		Analytics: mycConfig.AnalyticsConfig{Enabled: !m.Spec.DisableAnalytics},
		Web: mycConfig.WebConfig{
			WebDirectory:    "/ui",
			EnableProfiling: false,
			Http: mycConfig.HttpConfig{
				Enabled:     true,
				BindAddress: "0.0.0.0",
				Port:        8080,
			}},
		Bus: mycCmap.CustomMap{
			"type":         "embedded",
			"topic_prefix": "mc_server",
		},
		Logger: mycConfig.LoggerConfig{
			Mode:     "record_all",
			Encoding: "console",
			Level: mycConfig.LogLevelConfig{
				Core:       m.Spec.LogLevel,
				WebHandler: m.Spec.LogLevel,
				Storage:    "warn",
				Metric:     "warn",
			},
		},
		Directories: mycConfig.Directories{
			Data:          "/mc_home/data",
			Logs:          "/mc_home/logs",
			Tmp:           "/mc_home/tmp",
			SecureShare:   "/mc_home/secure_share",
			InsecureShare: "/mc_home/insecure_share",
		},
		Database: mycConfig.Database{
			Storage: mycCmap.CustomMap{
				"type":          "memory",
				"dump_enabled":  "true",
				"dump_interval": "10m",
				"dump_dir":      "memory_db",
				"dump_format":   []string{"yaml"},
				"load_format":   "yaml",
			},
			Metric: mycCmap.CustomMap{
				"disabled":             "false",
				"type":                 "influxdb",
				"uri":                  "http://localhost:8086",
				"token":                "",
				"username":             "",
				"password":             "",
				"organization_name":    "",
				"bucket_name":          "mycontroller",
				"batch_size":           "",
				"flush_interval":       "1s",
				"query_client_version": "",
			},
		},
		Gateway: mycServicefilter.ServiceFilter{
			Disabled: false,
			MatchAll: false,
			Types:    []string{},
			IDs:      []string{},
			Labels:   mycCmap.CustomStringMap{"location": "server"},
		},
		Handler: mycServicefilter.ServiceFilter{
			Disabled: false,
			MatchAll: false,
			Types:    []string{},
			IDs:      []string{},
			Labels:   mycCmap.CustomStringMap{"location": "server"},
		},
	}

	yamlBytes, err := yaml.Marshal(mycCfg)
	if err != nil {
		log.Error(err, "failed to convert config to yaml format", "ConfigMap.Namespace", m.Namespace, "ConfigMap.Name", m.Name)
		return err
	}

	cMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},

		Data: map[string]string{
			"mycontroller.yaml": string(yamlBytes),
		},
	}
	ctrl.SetControllerReference(m, cMap, r.Scheme)

	err = r.Create(ctx, cMap)
	if err != nil {
		log.Error(err, "failed to create new ConfigMap", "Namespace", cMap.Namespace, "Name", cMap.Name)
		return err
	}
	return nil
}

func (r *MyControllerReconciler) createService(ctx context.Context, m *mycontrollerv1.MyController) error {
	log := ctrllog.FromContext(ctx)

	mycService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labelsForMyController(m.Name),
			Ports:    []corev1.ServicePort{{Name: "http-web", Port: 8080}},
		},
	}
	ctrl.SetControllerReference(m, mycService, r.Scheme)
	err := r.Create(ctx, mycService)
	if err != nil {
		log.Error(err, "failed to create new service", "Namespace", mycService.Namespace, "Name", mycService.Name)
		return err
	}

	return nil
}

func (r *MyControllerReconciler) createRoute(ctx context.Context, m *mycontrollerv1.MyController) error {
	// if it is a openshift platform continue or exit
	if !isOpenshift() {
		return nil
	}

	log := ctrllog.FromContext(ctx)

	mycRoute := &routev1.Route{ObjectMeta: metav1.ObjectMeta{
		Name:      m.Name,
		Namespace: m.Namespace,
	},
		Spec: routev1.RouteSpec{
			To:   routev1.RouteTargetReference{Kind: "Service", Name: m.Name},
			Port: &routev1.RoutePort{TargetPort: intstr.FromString("http-web")},
			TLS:  &routev1.TLSConfig{},
		},
	}
	// Set MyController instance as the owner and controller
	ctrl.SetControllerReference(m, mycRoute, r.Scheme)

	err := r.Create(ctx, mycRoute)
	if err != nil {
		log.Error(err, "failed to create new route", "Namespace", mycRoute.Namespace, "Name", mycRoute.Name)
		return err
	}

	return nil
}

func labelsForMyController(name string) map[string]string {
	return map[string]string{"app": "mycontroller", "mycontroller_cr": name}
}

/*
Copyright 2022.

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
	"k8s.io/apimachinery/pkg/types"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	volcv1alpha1 "bytedance.com/mysql/api/v1alpha1"
)

// MySQLReconciler reconciles a MySQL object
type MySQLReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=volc.bytedance.com,resources=mysqls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=volc.bytedance.com,resources=mysqls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=volc.bytedance.com,resources=mysqls/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MySQL object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *MySQLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	log := log.FromContext(ctx)

	mysql := &volcv1alpha1.MySQL{}
	if err := r.Get(ctx, req.NamespacedName, mysql); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var replica int32 = 1
	var terminationGracePeriodSeconds int64 = 10
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysql.Name,
			Namespace: mysql.Namespace,
		},
		Type: "opaque",
		StringData: map[string]string{
			"MYSQL_ROOT_PASSWORD": "bytedance",
		},
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysql.Name,
			Namespace: mysql.Namespace,
			Labels: map[string]string{
				"app": mysql.Name,
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port: 3306,
				},
			},
			Selector: map[string]string{
				"app": mysql.Name,
			},
		},
	}
	statefulset := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysql.Name,
			Namespace: mysql.Namespace,
		},
		Spec: appv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": mysql.Name,
				},
			},
			ServiceName: service.Name,
			Replicas:    &replica,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": mysql.Name,
					},
				},
				Spec: v1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []v1.Container{
						{
							Name:  mysql.Name,
							Image: "flyer103/mysql:" + mysql.Spec.Version,
							Ports: []v1.ContainerPort{
								{
									HostPort: 3306,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      mysql.Name + "-store",
									MountPath: "/var/lib/mysql",
								},
							},
							Env: []v1.EnvVar{
								{
									Name: "MYSQL_ROOT_PASSWORD",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "mysql-password",
											},
											Key: "MYSQL_ROOT_PASSWORD",
										},
									},
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: mysql.Name + "-store",
					},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{
							"ReadWriteOnce",
						},
					},
				},
			},
		},
	}

	myselNamespacedName := types.NamespacedName{
		Name:      mysql.Name,
		Namespace: mysql.Namespace,
	}
	getErr := r.Client.Get(context.TODO(), myselNamespacedName, secret)
	if getErr != nil && getErr.Error() != "NotFound" {
		return ctrl.Result{}, getErr
	}
	if getErr == nil {
		// add function
		log.Info("creating secret")
		err := r.Client.Create(context.TODO(), secret)
		if err != nil {
			log.Info(err.Error())
			mysql.Status.Code = "failed"
			mysql.Status.Message = "failed"
			return ctrl.Result{}, err
		}
		log.Info("created secret")
	} else {
		// update function
		log.Info("updating secret")
		err := r.Client.Update(context.TODO(), secret)
		if err != nil {
			log.Info(err.Error())
			mysql.Status.Code = "failed"
			mysql.Status.Message = "failed"
			return ctrl.Result{}, err
		}
		log.Info("updated secret")
	}

	getErr = r.Client.Get(context.TODO(), myselNamespacedName, service)
	if getErr != nil && getErr.Error() != "NotFound" {
		return ctrl.Result{}, getErr
	}
	if getErr == nil {
		// add function
		log.Info("creating service")
		err := r.Client.Create(context.TODO(), service)
		if err != nil {
			log.Info(err.Error())
			mysql.Status.Code = "failed"
			mysql.Status.Message = "failed"
			return ctrl.Result{}, err
		}
		log.Info("created service")
	} else {
		// update function
		log.Info("updating service")
		err := r.Client.Update(context.TODO(), service)
		if err != nil {
			log.Info(err.Error())
			mysql.Status.Code = "failed"
			mysql.Status.Message = "failed"
			return ctrl.Result{}, err
		}
		log.Info("updated service")
	}

	getErr = r.Client.Get(context.TODO(), myselNamespacedName, statefulset)
	if getErr != nil && getErr.Error() != "NotFound" {
		return ctrl.Result{}, getErr
	}
	if getErr == nil {
		// add function
		log.Info("creating statefulset")
		err := r.Client.Create(context.TODO(), statefulset)
		if err != nil {
			log.Info(err.Error())
			mysql.Status.Code = "failed"
			mysql.Status.Message = "failed"
			return ctrl.Result{}, err
		}
		log.Info("created statefulset")
	} else {
		// update function
		log.Info("updating statefulset")
		err := r.Client.Update(context.TODO(), statefulset)
		if err != nil {
			log.Info(err.Error())
			mysql.Status.Code = "failed"
			mysql.Status.Message = "failed"
			return ctrl.Result{}, err
		}
		log.Info("updated statefulset")
	}
	mysql.Status.Code = "success"
	mysql.Status.Message = "success"
	// delete function
	if mysql.Status.Message == "deleting" {
		log.Info("deleting mysql")
		err := r.Client.Delete(context.TODO(), secret)
		if err != nil {
			log.Info(err.Error())
			return ctrl.Result{}, err
		}
		log.Info("deleted secret")

		err = r.Client.Delete(context.TODO(), service)
		if err != nil {
			log.Info(err.Error())
			return ctrl.Result{}, err
		}
		log.Info("deleted service")

		err = r.Client.Delete(context.TODO(), statefulset)
		if err != nil {
			log.Info(err.Error())
			return ctrl.Result{}, err
		}
		log.Info("deleted statefulset")

		err = r.Client.Delete(context.TODO(), mysql)
		if err != nil {
			log.Info(err.Error())
			return ctrl.Result{}, err
		}
		log.Info("deleted mysql")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MySQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&volcv1alpha1.MySQL{}).
		Complete(r)
}

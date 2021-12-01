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
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clustersecurityiov1 "github.com/escoffier/cluster-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const manageredClusterFinalizer = "cluster.security.io/api-resource-cleanup"

// ManagedClusterReconciler reconciles a ManagedCluster object
type ManagedClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cluster.security.io.github.com,resources=managedclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster.security.io.github.com,resources=managedclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.security.io.github.com,resources=managedclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ManagedClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile clusters")

	// TODO(user): your logic here
	managedCluster := &clustersecurityiov1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "",
		},
	}
	err := r.Get(ctx, req.NamespacedName, managedCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("cluster is deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "no managered cluster")
		return ctrl.Result{}, err
	}

	managedCluster = managedCluster.DeepCopy()
	if !managedCluster.DeletionTimestamp.IsZero() {
		logger.Info("finalizer ")
		// secret := &corev1.Secret{}
		// err = r.Get(ctx, types.NamespacedName{Namespace: managedCluster.Namespace, Name: managedCluster.Name}, secret)
		// if err == nil {
		// 	logger.Info("deleteing secret")
		// 	logger.Info(secret.Name)
		// 	err = r.Delete(ctx, secret)
		// 	if err != nil {
		// 		logger.Error(err, "failed to delete secret")
		// 		return ctrl.Result{}, err
		// 	}
		// }
		err = r.removeManagedClusterFinalizer(ctx, managedCluster)
		if err != nil {
			logger.Error(err, "failed to delete finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// config := rest.Config{
	// 	BearerToken: managedCluster.Spec.ClientConfig.BearToken,
	// 	TLSClientConfig: rest.TLSClientConfig{
	// 		CAData:   managedCluster.Spec.ClientConfig.CAData,
	// 		CertData: managedCluster.Spec.ClientConfig.CertData,
	// 		KeyData:  managedCluster.Spec.ClientConfig.KeyData,
	// 	},
	// }
	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Namespace: managedCluster.Namespace, Name: managedCluster.Name}, secret)

	if err != nil {
		if errors.IsNotFound(err) {
			data, err := json.Marshal(&managedCluster.Spec.ClientConfig)
			if err != nil {
				logger.Error(err, "no managered cluster")
				return ctrl.Result{}, err
			}
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedCluster.Name,
					Namespace: managedCluster.Namespace,
				},
				Data: map[string][]byte{"config": data},
			}
			logger.Info("creating secret")
			ctrl.SetControllerReference(managedCluster, secret, r.Scheme)
			err = r.Create(ctx, secret)
			if err != nil {
				logger.Error(err, "failed to create secret")
				return ctrl.Result{}, err
			}
			// ctrl.SetControllerReference(managedCluster, secret, r.Scheme)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "get secret error")
		return ctrl.Result{}, err
	}
	logger.Info("secret already existed")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersecurityiov1.ManagedCluster{}).
		Owns(&corev1.Secret{}).
		// Owns(&corev1.Secret{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
		// 	fmt.Printf("%s-%s\n", object.GetNamespace(), object.GetName())
		// 	return true
		// }))).
		Complete(r)
}

func (r *ManagedClusterReconciler) removeManagedClusterFinalizer(ctx context.Context, manageredCluster *clustersecurityiov1.ManagedCluster) error {
	cpyFinalizer := []string{}
	for _, f := range manageredCluster.Finalizers {
		if f == manageredClusterFinalizer {
			continue
		}
		cpyFinalizer = append(cpyFinalizer, f)
	}
	if len(cpyFinalizer) != len(manageredCluster.Finalizers) {
		manageredCluster.Finalizers = cpyFinalizer
		err := r.Update(ctx, manageredCluster)
		return err
	}
	return nil
}

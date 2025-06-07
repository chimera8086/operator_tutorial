/*
Copyright 2025.

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

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tutorialv1alpha1 "operator_tutorial/secsynch/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecSynchReconciler reconciles a SecSynch object
type SecSynchReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tutorial.study.dev,resources=secsynches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tutorial.study.dev,resources=secsynches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tutorial.study.dev,resources=secsynches/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SecSynch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
func (r *SecSynchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// в переменной req находится namespace/имя cd, который вызвал срабатывание Reconcile

	log := log.FromContext(ctx).WithValues("secsynch_tutorial", req.NamespacedName)
	log.Info("Reconsiller secsynch_tutorial start")

	// в r.Get передается имя Неймспейс/Имя ресурса (SecSynch), на который сработал Reconcile
	// если контроллер только запустился, то он получает список всех SecSynch в кластере и пытается применить бизнес логику

	// в r.Get тип ресурса (Pod, Deployment, Job) определяется по переменной, в которую надо записать результат, в нашем случае cr
	cr := &tutorialv1alpha1.SecSynch{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// ситуация когда ресурс не найден в кластере - штатная, вызывать ошибку не нужно
			log.Info("Resource SecSynch not found, then it usually means that it was deleted")
			return ctrl.Result{}, nil
		}
		// в ином случае это нештатная ситуация и ошибка.
		// контроллер будет сыпать в логи ошибку и номер строки, на которой она произошла
		// так же кластер будет заново вызывать функцию Reconcile до победного конца.
		// Время между запросов увеличивается от 5 миллисекунд до 1000 секунд по экспоненте
		log.Error(err, "Failed to get SecSynch ")
		return ctrl.Result{}, err
	}

	// получаем секрет который надо копировать
	sourceSecret := &corev1.Secret{}
	// req.NamespacedName такое же тип объекта как и types.NamespacedName{Namespace: cr.Spec.SourceNamespace, Name: cr.Spec.SecretName}
	err = r.Get(ctx, types.NamespacedName{Namespace: cr.Spec.SourceNamespace, Name: cr.Spec.SecretName}, sourceSecret)
	if err != nil {
		// штатный случай, просто нет секрета в кластере
		if errors.IsNotFound(err) {
			log.Info("Resource Secret not found, SecretName:", cr.Spec.SecretName, "SourceNamespace", cr.Spec.SourceNamespace, "try again in 3 minutes")
			// передаем команду - вызвать повторно функцию Reconcile
			// через 3 минуты, может тогда уже будет существовать секрет
			return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
		}
	}
	// перебираем ns в которых должен быть секрет
	for _, destNS := range cr.Spec.DestinationNamespaces {
		// запрашиваем секрет в ns куда надо его скопировать, вдруг он там уже есть.
		destSecret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Namespace: destNS, Name: cr.Spec.SecretName}, destSecret)
		// секрет в ns есть, ничего не делаем
		if err == nil {
			continue
		}
		// секрета в ns нет, создаем его
		if errors.IsNotFound(err) {
			log.Info("Creating Secret ", "destination namespace  is", destNS)
			destSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sourceSecret.Name,
					Namespace: destNS,
				},
				// копируем содержимое секрета
				Data: sourceSecret.Data,
			}

			err_create := r.Create(ctx, destSecret)
			// ситуация когда ns еще нет и из-за этого не может создать секрет, пробуем позже
			if errors.IsNotFound(err_create) {
				log.Info("Namespace not found", "Namespace is", destNS)
				return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil

			}
			if err_create != nil {
				log.Error(err_create, "Secret is not created in", " Namespace is ", destNS, " SecretName is ", sourceSecret.Name)
				return ctrl.Result{}, err_create
			}
			continue

		}
		// не смогли получить секрет, ошибка не из-за отсутствие объекта
		if err != nil {
			log.Error(err, "Not get secret ", "Name is", sourceSecret.Name, "in Namespace", destNS)
			return ctrl.Result{}, err
		}

	}
	// обновляем время успешной синхронизации секрета
	cr.Status.LastSyncTime = metav1.Now()
	if err := r.Status().Update(ctx, cr); err != nil {
		log.Error(err, "Unable to update secretsync status")
		return ctrl.Result{}, err
	}
	log.Info("Status secretsync updated", "LastSyncTime", cr.Status.LastSyncTime)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecSynchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tutorialv1alpha1.SecSynch{}).
		Named("secsynch").
		Complete(r)
}

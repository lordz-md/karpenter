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

package debug

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1beta1 "github.com/aws/karpenter-core/pkg/apis/v1beta1"
	corecontroller "github.com/aws/karpenter-core/pkg/operator/controller"
)

type NodeClaimController struct {
	kubeClient client.Client
}

func NewNodeClaimController(kubeClient client.Client) *NodeClaimController {
	return &NodeClaimController{
		kubeClient: kubeClient,
	}
}

func (c *NodeClaimController) Name() string {
	return "nodeclaim"
}

func (c *NodeClaimController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	nc := &corev1beta1.NodeClaim{}
	if err := c.kubeClient.Get(ctx, req.NamespacedName, nc); err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("[DELETED %s] NODECLAIM %s\n", time.Now().Format(time.RFC3339), req.NamespacedName.String())
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Printf("[CREATED/UPDATED %s] NODECLAIM %s %s\n", time.Now().Format(time.RFC3339), req.NamespacedName.Name, c.GetInfo(nc))
	return reconcile.Result{}, nil
}

func (c *NodeClaimController) GetInfo(nc *corev1beta1.NodeClaim) string {
	return fmt.Sprintf("ready=%t launched=%t registered=%t initialized=%t",
		nc.StatusConditions().IsHappy(),
		nc.StatusConditions().GetCondition(corev1beta1.Launched).IsTrue(),
		nc.StatusConditions().GetCondition(corev1beta1.Registered).IsTrue(),
		nc.StatusConditions().GetCondition(corev1beta1.Initialized).IsTrue(),
	)
}

func (c *NodeClaimController) Builder(_ context.Context, m manager.Manager) corecontroller.Builder {
	return corecontroller.Adapt(controllerruntime.
		NewControllerManagedBy(m).
		For(&corev1beta1.NodeClaim{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldNodeClaim := e.ObjectOld.(*corev1beta1.NodeClaim)
				newNodeClaim := e.ObjectNew.(*corev1beta1.NodeClaim)
				return c.GetInfo(oldNodeClaim) != c.GetInfo(newNodeClaim)
			},
		}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}))
}

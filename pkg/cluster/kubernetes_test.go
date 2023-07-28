// MIT License
//
// Copyright (c) 2023 kache.io
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// appsv1 "k8s.io/api/apps/v1"
	// v1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/util/wait"
	// "k8s.io/apimachinery/pkg/watch"
	// "k8s.io/client-go/informers"
	// "k8s.io/client-go/kubernetes/fake"
	// clienttesting "k8s.io/client-go/testing"
	// "k8s.io/client-go/tools/cache"
)

func ATestCluster(t *testing.T) {
	c, err := NewKubernetesClient("default", "kache-service")
	require.NoError(t, err)

	assert.Equal(t, 3, len(c.Endpoints("api")))
}

// func TestKubeClient(t *testing.T) {

// 	replicas := int32(2)

// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "kache",
// 		},
// 		Spec: appsv1.DeploymentSpec{
// 			Replicas: &replicas,
// 			Selector: &metav1.LabelSelector{
// 				MatchLabels: map[string]string{
// 					"app": "kache",
// 				},
// 			},
// 			Template: v1.PodTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Labels: map[string]string{
// 						"app": "kache",
// 					},
// 				},
// 				Spec: v1.PodSpec{
// 					Containers: []v1.Container{
// 						{
// 							Name:  "kache",
// 							Image: "kache/kache",
// 							Ports: []v1.ContainerPort{
// 								{
// 									Name:          "api",
// 									Protocol:      v1.ProtocolTCP,
// 									ContainerPort: 1338,
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	service := &v1.Service{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "kache-service",
// 			Namespace: "test",
// 			Labels: map[string]string{
// 				"app": "kache",
// 			},
// 		},
// 		Spec: v1.ServiceSpec{
// 			Ports:     nil,
// 			Selector:  nil,
// 			ClusterIP: "",
// 		},
// 	}

// 	// Create the fake client.
// 	client := fake.NewSimpleClientset()

// 	var err error
// 	_, err = client.AppsV1().Deployments("test").Create(context.TODO(), deployment, metav1.CreateOptions{})
// 	require.NoError(t, err)

// 	_, err = client.CoreV1().Services("test").Create(context.TODO(), service, metav1.CreateOptions{})
// 	require.NoError(t, err)

// 	time.Sleep(4 * time.Second)

// 	c, err := NewClient()
// 	require.NoError(t, err)

// 	assert.Equal(t, 3, len(c.Endpoints("test", "kache-service", "api")))
// }

// // TestFakeClient demonstrates how to use a fake client with SharedInformerFactory in tests.
// func TestFakeClient(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	watcherStarted := make(chan struct{})
// 	// Create the fake client.
// 	client := fake.NewSimpleClientset()
// 	// A catch-all watch reactor that allows us to inject the watcherStarted channel.
// 	client.PrependWatchReactor("*", func(action clienttesting.Action) (handled bool, ret watch.Interface, err error) {
// 		gvr := action.GetResource()
// 		ns := action.GetNamespace()
// 		watch, err := client.Tracker().Watch(gvr, ns)
// 		if err != nil {
// 			return false, nil, err
// 		}
// 		close(watcherStarted)
// 		return true, watch, nil
// 	})

// 	// We will create an informer that writes added pods to a channel.
// 	pods := make(chan *v1.Pod, 1)
// 	informers := informers.NewSharedInformerFactory(client, 0)
// 	podInformer := informers.Core().V1().Pods().Informer()
// 	podInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(obj interface{}) {
// 			pod := obj.(*v1.Pod)
// 			t.Logf("pod added: %s/%s", pod.Namespace, pod.Name)
// 			pods <- pod
// 		},
// 	})

// 	// Make sure informers are running.
// 	informers.Start(ctx.Done())

// 	// This is not required in tests, but it serves as a proof-of-concept by
// 	// ensuring that the informer goroutine have warmed up and called List before
// 	// we send any events to it.
// 	cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced)

// 	// The fake client doesn't support resource version. Any writes to the client
// 	// after the informer's initial LIST and before the informer establishing the
// 	// watcher will be missed by the informer. Therefore we wait until the watcher
// 	// starts.
// 	// Note that the fake client isn't designed to work with informer. It
// 	// doesn't support resource version. It's encouraged to use a real client
// 	// in an integration/E2E test if you need to test complex behavior with
// 	// informer/controllers.
// 	<-watcherStarted
// 	// Inject an event into the fake client.
// 	p := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "kache-pod"}}
// 	_, err := client.CoreV1().Pods("kache-ns").Create(context.TODO(), p, metav1.CreateOptions{})
// 	if err != nil {
// 		t.Fatalf("error injecting pod add: %v", err)
// 	}

// 	select {
// 	case pod := <-pods:
// 		t.Logf("Got pod from channel: %s/%s", pod.Namespace, pod.Name)
// 	case <-time.After(wait.ForeverTestTimeout):
// 		t.Error("Informer did not get the added pod")
// 	}

// 	c, err := NewClient()
// 	require.NoError(t, err)

// 	assert.Equal(t, 3, len(c.Endpoints("", "kache-service", "api")))
// }

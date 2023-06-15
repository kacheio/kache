## Start a Kubernetes cluster using minikube

or with Docker Desktop. 

```
$ minikube start
```

## Deploy Redis

```
$ kubectl apply -f deployments/redis-master.yml
deployment.apps/redis-master created
service/redis-master create`

$ kubectl get pods
NAME                            READY   STATUS    RESTARTS   AGE
redis-master-7b44998456-pl8h9   1/1     Running   0          34s
```

## Deploy the kache

```
$ kubectl apply -f cloud/kubernetes/kache.yml
deployment.apps/kache-app created
service/kache-app-service created
````

```
$ kubectl get pods
NAME                            READY   STATUS    RESTARTS   AGE
kache-app-57b7d4d4cd-fkddw   1/1     Running   0          27s
kache-app-57b7d4d4cd-l9wg9   1/1     Running   0          27s
kache-app-57b7d4d4cd-m9t8b   1/1     Running   0          27s
redis-master-7b44998456-pl8h9   1/1     Running   0          82s
````

## Accessing the application

The Go app is exposed as NodePort via the service. You can get the service URL using minikube like this -

```
$ minikube service kache-app-service --url
http://192.168.99.100:30435
```

You can use the above endpoint to access the application:

```
$ curl http://192.168.99.100:30435
...
```

## Kubectl commands

kubectl get nodes -o wide  

kubectl get endpoints   

kubectl get pods 

kubectl get svc  



kubectl apply -f cloud/kubernetes/kache.yml 

kubectl rollout restart deployment

kubectl exec kache-7d644995dd-fw8b5 -- cat /etc/kache/config.yml 

 kubectl create configmap kache-config --from-file=cloud/kubernetes/configmap.yml 
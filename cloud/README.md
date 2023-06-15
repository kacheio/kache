# Kache on Kubernetes

The following are instructions for launching kache on kubernetes. 

:warning: Please note that this is not intended for use in a production environment. We will provide more sophisticated configurations for operations in the future. 

## Getting started

The following describes how to run kache on a local Kubernetes cluster.

### Start a Kubernetes cluster

To start a cluster, use minikube or Docker Desktop.

```
minikube start
```

### Create ConfigMap

Create a ConfigMap that contains a the kache configuration:

```
kubectl create configmap kache-config --from-file=cloud/kubernetes/configmap.yml 
```

Apply the ConfigMap:

```
kubectl apply -f cloud/kubernetes/configmap.yml
```

### Deploy Redis

```
kubectl apply -f cloud/kubernetes/redis-master.yml
```

### Deploy Kache

```
kubectl apply -f cloud/kubernetes/kache.yml
````

### Accessing the service

Check that the Pods are up and running:

```
$ kubectl get pods 

NAME                           READY   STATUS    RESTARTS   AGE
kache-54cd8ffd96-xzdqg         1/1     Running   0          14h
redis-master-d4f785667-mpmvg   1/1     Running   0          14h
```

The Kache service is exposed as a LoadBalancer via the service with mapped ports and is accessible on localhost.

```
$ kubectl get svc

NAME            TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                                      AGE
kache-service   LoadBalancer   10.110.92.73   localhost     80:30135/TCP,1337:32284/TCP,1338:30691/TCP   44h
kubernetes      ClusterIP      10.96.0.1      <none>        443/TCP                                      44h
redis-master    ClusterIP      10.97.188.34   <none>        6379/TCP                                     44h
```

Use the above endpoints to access the service:

```
curl http://localhost:1337/
```

Access the API:

```
curl http://localhost:1337/api/v1/
```

## Helm Chart installation

TODO: Use the Helm chart to rollout an instance of Kache.

## Build and run with Docker 

Build your own image locally:

```
docker build -t $IMAGE_NAME -f Dockerfile .
```

Alternatively, use the official [Docker image](https://hub.docker.com/r/kacheio/kache) and run it with the sample configuration file:

```
docker run -it -p 80:80 -v $PWD/kache.yml:/etc/kache/kache.yml kache -config.file=/etc/kache/kache.yml 
````

If you want to run Kache with a distributed caching backend (e.g. Redis), you can use and run this example [docker-compose](https://github.com/kacheio/kache/blob/main/cloud/docker/docker-compose.yml) as a starting point:

```
docker-compose -f cloud/docker/docker-compose.yml up 
```

## Troubleshooting

If there are problems loading the configuration, verify that the Pod has the latest configuration:
```
kubectl exec $POD_NAME -- cat /etc/kache/config.yml 
```

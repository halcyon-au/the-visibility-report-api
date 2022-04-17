# Deployment
Deployment is done through kubernetes, it consists of a deploy & svc of mongodb, api and worker.

## GitHub Packages Authentication
1. Create A Github PAT And Base64 it with username  
``echo -n "username:123123adsfasdf123123" | base64``
2. Create a JSON string with the following format
```
{
    "auths":
    {
        "ghcr.io":
            {
                "auth":"<The Base 64 String From Previous Step>"
            }
    }
}
```
3. Base64 Encode the previous json string  
``echo -n  '<JSON STRING>' | base64``
4. Store it in a secrets file with the format { .dockerconfigjson: \<BASE64 JSON\> } and create secret with name  
``kubectl create -f dockerconfigjson.yaml``

## Local Testing

You can use [Minikube](https://minikube.sigs.k8s.io/docs) to run the cluster locally and test it out.

### Start Minikube
``minikube start``
### Reset Minikube
``minikube delete``
``minikube start``
### Create A Tunnel
NOTE: Do this in a spare terminal window  
``minikube tunnel``

## MongoDB

Dependencies: Secrets created with the format { MONGO_INITDB_ROOT_USERNAME, MONGO_INITDB_ROOT_PASSWORD } saved as mongodb-secret.yaml
1. Create the secrets with name mongodb-secret  
``kubectl create -f mongodb-secrets.yaml``
2. Create the persistent volume  
``kubectl create -f mongo-pvc.yaml``
3. Create the deployment    
``kubectl create -f mongo-deployment.yaml``
4. Create the service to expose it internally  
``kubectl create -f mongo-service.yaml``
5. 🔧 Test It 🔧
```
kubectl get pods -> Get Pod name of Mongodb
kubectl exec -it <PODNAME> /bin/bash
mongo -u $MONGO_INITDB_ROOT_USERNAME -p $MONGO_INITDB_ROOT_PASSWORD
```

## API

Dependencies: Secrets created with the format { mongousername, mongopassword } saved as api-secrets.yaml and github auth setup
1. Create the github secrets with the name dockerconfigjson-github-com  
``kubectl create -f githubpackages-secrets.yaml``
2. Create the secrets with name api-secrets  
``kubectl create -f api-secrets.yaml``
3. Create the api deployment  
``kubectl create -f api-deployment.yaml``
4. Expose the api deployment  
``kubectl expose deployment visibilityreportapi --type=LoadBalancer --name=visibilityreport-svc``
5. 🔧 Test It 🔧  
``curl http://<kubernetes_endpoint>:1323/api/v1/hb``

## Worker

Dependencies: Secrets created with the format { mongousername, mongopassword } saved as api-secrets.yaml and github auth setup
1. Create the github secrets with the name dockerconfigjson-github-com  
``kubectl create -f githubpackages-secrets.yaml``
2. Create the secrets with name api-secrets  
``kubectl create -f api-secrets.yaml``
3. Create the worker deployment
``kubectl create -f worker-deployment.yaml``
5. 🔧 Test It 🔧  
``kubectl logs deployment/visibilityreportapi-worker``
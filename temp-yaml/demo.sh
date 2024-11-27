#!/bin/bash env

set -x

kubectl apply -f 07d_demo_pod_with_secret.yaml -f 07c_demo_secret.yaml -f cr.yaml

echo "Waiting for the reconciler to run for all pods and the cr..."

sleep 15

kubectl get immutableimages.batch.github.com immutable-secret-image-list \
        -o jsonpath='{.spec.imageSecretMap}'

echo "Check kubectl if immutable added to the secrets"
kubectl get pods pod-1 -o jsonpath='{.spec.containers[0].image}'
kubectl get secrets secret-1 -o yaml | grep immutable
kubectl get pods pod-2 -o jsonpath='{.spec.containers[0].image}'
kubectl get secrets secret-2 -o yaml | grep immutable
kubectl get pods pod-image-not-in-cr -o jsonpath='{.spec.containers[0].image}'
kubectl get secrets secret-image-not-in-cr -o yaml | grep immutable
kubectl get secrets secret-added-later -o yaml | grep immutable
kubectl apply -f test-pod.yaml
echo "Waiting for the reconciler to run..."
sleep 2
kubectl get secrets secret-added-later -o yaml | grep immutable

echo "Trying to edit a secret that should be immutable"
newpass=$(echo 'newpass' | base64); kubectl patch secret secret-1 -p "{\"data\": {\"password.txt\": \"$newpass\"}}"
read -p "delete custom resource?(y/n) " yn
case $yn in 
	  y ) echo "Check which secrets added to imagemap";
        kubectl get immutableimages.batch.github.com immutable-secret-image-list \
                -o jsonpath='{.spec.imageSecretMap}';
        echo Deleting;
          kubectl delete immutableimages.batch.github.com immutable-secret-image-list;;
	  n ) echo Not deleting CR...;;
	  * ) echo invalid response;
		    exit 1;;
esac

echo "Now that the immutableimages list is gone, we can edit the secret"
newpass=$(echo 'newpass' | base64); kubectl patch secret secret-1 -p "{\"data\": {\"password.txt\": \"$newpass\"}}"

echo "The updated password is "
kubectl get secrets secret-1 --template='{{ range $key, $value := .data }}{{ printf "%s: %s\n" $key ($value | base64decode) }}{{ end }}'


read -p "delete all resources?(y/n) " yn


case $yn in 
	  y ) echo Deleting;
          kubectl delete secrets secret-1 secret-2 secret-image-not-in-cr secret-added-later;
          kubectl delete pods pod-1 pod-2 pod-image-not-in-cr pod-added-later;
          kubectl delete immutableimages.batch.github.com immutable-secret-image-list 2>/dev/null;
          exit;;
	  n ) echo exiting...;
		     exit;;
	  * ) echo invalid response;
		    exit 1;;
esac

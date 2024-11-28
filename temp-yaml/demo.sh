#!/bin/bash env

add_separator () {
    { set +x; } 2>/dev/null
    sleep 2
    echo "------------------------------------------------"
    echo "------------------------------------------------"
    set -x
}

set -x

kubectl apply -f 07d_demo_pod_with_secret.yaml -f 07c_demo_secret.yaml -f cr.yaml

{ set +x; } 2>/dev/null
echo "Waiting for the reconciler to run for all pods and the cr..."
sleep 10
set -x

read -rp "Continue?(y/n) " yn
{ set +x; } 2>/dev/null
case $yn in 
	  y ) echo "Proceeding";;
    n ) echo "Proceeding";;
    * ) echo "Invalid response";;
esac
set -x

{ add_separator; } 2>/dev/null

kubectl get immutableimages.batch.github.com immutable-secret-image-list \
        -o jsonpath='{.spec.imageSecretMap}'
{ set +x; } 2>/dev/null
echo 
set -x
kubectl get pods pod-1 -o jsonpath='{.spec.containers[0].image}'
{ set +x; } 2>/dev/null
echo 
set -x
kubectl get pods pod-2 -o jsonpath='{.spec.containers[0].image}'
{ set +x; } 2>/dev/null
echo 
set -x
kubectl get pods pod-image-not-in-cr -o jsonpath='{.spec.containers[0].image}'
{ set +x; } 2>/dev/null
echo 
set -x

kubectl get immutableimages.batch.github.com immutable-secret-image-list \
        -o jsonpath='{.spec.imageSecretMap}'
{ set +x; } 2>/dev/null
echo 

{ add_separator; } 2>/dev/null

kubectl apply -f test-pod.yaml
{ set +x; } 2>/dev/null
echo "Adding pod pod-added-later"
echo "Waiting for the reconciler to run..."
sleep 4
set -x
kubectl get immutableimages.batch.github.com immutable-secret-image-list \
        -o jsonpath='{.spec.imageSecretMap}'
{ set +x; } 2>/dev/null
echo 

{ add_separator; } 2>/dev/null

{ set +x; } 2>/dev/null
echo "Trying to edit a secret that should be immutable"
set -x
newpass=$(echo 'newpass' | base64); kubectl patch secret secret-1 -p "{\"data\": {\"password.txt\": \"$newpass\"}}"

{ add_separator; } 2>/dev/null


read -rp "Update custom resource?(y/n) [Edit cr.yaml]" yn
case $yn in 
	  y ) echo "Check which secrets added to imagemap";
        kubectl get immutableimages.batch.github.com immutable-secret-image-list \
                -o jsonpath='{.spec.imageSecretMap}';
        kubectl patch immutableimages.batch.github.com immutable-secret-image-list \
                --type='json' \
                -p '[{"op": "remove", "path": "/spec/imageSecretMap/alpine:latest", "value": []}]';
        echo Updated;;
          # kubectl delete immutableimages.batch.github.com immutable-secret-image-list;;
	  n ) echo Not updating CR...;;
	  * ) echo invalid response;
		    exit 1;;
esac

{ add_separator; } 2>/dev/null

{ set +x; } 2>/dev/null
echo "Now that the immutableimages list doesnt having alpine, we can edit the secret"
set -x
kubectl get immutableimages.batch.github.com immutable-secret-image-list \
        -o jsonpath='{.spec.imageSecretMap}';
{ set +x; } 2>/dev/null
echo 
set -x
newpass=$(echo 'newpass' | base64); kubectl patch secret secret-1 -p "{\"data\": {\"password.txt\": \"$newpass\"}}"

{ set +x; } 2>/dev/null 
echo "The updated password is "
set -x
kubectl get secrets secret-1 --template='{{ range $key, $value := .data }}{{ printf "%s: %s\n" $key ($value | base64decode) }}{{ end }}'

{ add_separator; } 2>/dev/null

read -rp "delete pod-added-later?(y/n) " yn
case $yn in 
	  y ) echo Deleting;
        kubectl delete pods pod-added-later;
        { set +x; } 2>/dev/null;
        echo "The secret-added-later should not be in CR now";
        set -x;
        kubectl get immutableimages.batch.github.com immutable-secret-image-list \
                -o jsonpath='{.spec.imageSecretMap}';
        { set +x; } 2>/dev/null;
        echo 
        echo "The edit to secret-added-later is successful";
        set -x;
        kubectl patch secret secret-added-later -p "{\"data\": {\"password.txt\": \"$newpass\"}}";
        { set +x; } 2>/dev/null;
        echo "The updated password is ";
        set -x;
        kubectl get secrets secret-added-later \
                --template='{{ range $key, $value := .data }}{{ printf "%s: %s\n" $key ($value | base64decode) }}{{ end }}';;
	  n ) echo Not deleting pod...;;
	  * ) echo invalid response;
		    exit 1;;
esac

{ add_separator; } 2>/dev/null

read -rp "delete all resources?(y/n) " yn


case $yn in 
	  y ) echo Deleting;
          kubectl delete secrets secret-1 secret-2 secret-image-not-in-cr secret-added-later;
          kubectl delete pods pod-1 pod-2 pod-image-not-in-cr;
          kubectl delete pods pod-added-later 2>/dev/null;
          kubectl delete immutableimages.batch.github.com immutable-secret-image-list 2>/dev/null;
          exit;;
	  n ) echo exiting...;
		     exit;;
	  * ) echo invalid response;
		    exit 1;;
esac

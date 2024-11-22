kubectl apply -f 07d_demo_pod_with_secret.yaml -f 07c_demo_secret.yaml -f cr.yaml

echo "All the pods, secrets and cr deployed"

echo "Waiting for the reconciler to run for all pods and the cr..."

sleep 10

echo "----------------------------"
echo "Check if secrets added to imagemap"
echo "kubectl get immutableimages.batch.github.com immutable-secret-image-list -o jsonpath='{.spec.imageSecretMap}'"
kubectl get immutableimages.batch.github.com immutable-secret-image-list \
        -o jsonpath='{.spec.imageSecretMap}'
printf "\n----------------------------\n"

echo "Check kubectl if immutable added to the secrets"
echo "----------------------------"

echo "kubectl get pods pod-1 -o jsonpath='{.spec.containers[0].image}'"
kubectl get pods pod-1 -o jsonpath='{.spec.containers[0].image}'
printf "\n----------------------------\n"
echo "kubectl get secrets secret-1 -o yaml | grep immutable"
kubectl get secrets secret-1 -o yaml | grep immutable
echo "----------------------------"
echo "kubectl get pods pod-2 -o jsonpath='{.spec.containers[0].image}'"
kubectl get pods pod-2 -o jsonpath='{.spec.containers[0].image}'
printf "\n----------------------------\n"
echo "kubectl get secrets secret-2 -o yaml | grep immutable"
kubectl get secrets secret-2 -o yaml | grep immutable
echo "----------------------------"
echo "kubectl get pods pod-image-not-in-cr -o jsonpath='{.spec.containers[0].image}'"
kubectl get pods pod-image-not-in-cr -o jsonpath='{.spec.containers[0].image}'
printf "\n----------------------------\n"
echo "For secret whose container image is not part of cr, its not tagged"
echo "kubectl get secrets secret-image-not-in-cr -o yaml | grep immutable"
kubectl get secrets secret-image-not-in-cr -o yaml | grep immutable
echo "----------------------------"
printf "\n\n"
echo "----------------------------"
echo "Check if the unused secret got tagged"
echo "kubectl get secrets secret-added-later -o yaml | grep immutable"
kubectl get secrets secret-added-later -o yaml | grep immutable
echo "----------------------------"

echo "Adding a new pod that uses secret-added-later"
echo "kubectl apply -f test-pod.yaml"
kubectl apply -f test-pod.yaml

echo "Waiting for the reconciler to run..."
sleep 2

echo "----------------------------"
echo "Now the unused secret is updated"
echo "kubectl get secrets secret-added-later -o yaml | grep immutable"
kubectl get secrets secret-added-later -o yaml | grep immutable
echo "----------------------------"


read -p "delete custom resource?(y/n)" yn
case $yn in 
	  y ) echo "Check which secrets added to imagemap";
        echo "kubectl get immutableimages.batch.github.com immutable-secret-image-list -o jsonpath='{.spec.imageSecretMap}'";
        kubectl get immutableimages.batch.github.com immutable-secret-image-list \
                -o jsonpath='{.spec.imageSecretMap}';
        printf "\n----------------------------\n";
        echo Deleting;
          kubectl delete immutableimages.batch.github.com immutable-secret-image-list;;
	  n ) echo Not deleting CR...;;
	  * ) echo invalid response;
		    exit 1;;
esac

echo "----------------------------"
echo "Immutable should be gone from the secrets"
echo "----------------------------"

echo "kubectl get secrets secret-1 -o yaml | grep immutable"
kubectl get secrets secret-1 -o yaml | grep immutable
echo "----------------------------"

echo "kubectl get secrets secret-2 -o yaml | grep immutable"
kubectl get secrets secret-2 -o yaml | grep immutable
echo "----------------------------"

echo "kubectl get secrets secret-added-later -o yaml | grep immutable"
kubectl get secrets secret-added-later -o yaml | grep immutable
echo "----------------------------"








read -p "delete all resources?(y/n)" yn


case $yn in 
	  y ) echo Deleting;
          kubectl delete secrets secret-1 secret-2 secret-image-not-in-cr secret-added-later;
          kubectl delete pods pod-1 pod-2 pod-image-not-in-cr pod-added-later;
          kubectl delete immutableimages.batch.github.com immutable-secret-image-list;;
	  n ) echo exiting...;
		     exit;;
	  * ) echo invalid response;
		    exit 1;;
esac

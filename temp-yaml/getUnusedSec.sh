envSecrets=$(kubectl get pods -o jsonpath='{.items[*].spec.containers[*].env[*].valueFrom.secretKeyRef.name}' | xargs -n1)
envSecrets2=$(kubectl get pods -o jsonpath='{.items[*].spec.containers[*].envFrom[*].secretRef.name}' | xargs -n1)
volumeSecrets=$(kubectl get pods -o jsonpath='{.items[*].spec.volumes[*].secret.secretName}' | xargs -n1)
pullSecrets=$(kubectl get pods -o jsonpath='{.items[*].spec.imagePullSecrets[*].name}' | xargs -n1)
tlsSecrets=$(kubectl get ingress -o jsonpath='{.items[*].spec.tls[*].secretName}' | xargs -n1)
SASecrets=$(kubectl get secrets --field-selector=type=kubernetes.io/service-account-token -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | xargs -n1)


diff \
<(echo "$envSecrets\n$envSecrets2\n$volumeSecrets\n$pullSecrets\n$tlsSecrets\n$SASecrets" | sort | uniq) \
<(kubectl get secrets -o jsonpath='{.items[*].metadata.name}' | xargs -n1 | sort | uniq)

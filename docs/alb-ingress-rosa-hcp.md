# Use AWS ALB as alternative ingress on ROSA HCP

This doc aims at showing the minimum efforts needed to expose the OpenShift console via AWS ALB on a ROSA HCP cluster.
The use case in mind is [HyperShift hosted clusters where the Ingress capability is disabled](https://github.com/openshift/enhancements/pull/1415).

## Requirements

- Ready ROSA HCP OpenShift cluster.
- [Installed AWS Load Balancer Operator and its controller](https://docs.openshift.com/rosa/networking/aws-load-balancer-operator.html).
- User logged as a cluster admin.

## Procedure

### Create certificate in AWS Certificate Manager

In order to configure an HTTPS listener on AWS ALB you need to have a certificate created in AWS Certificate Manager.
You can import an existing certificate or request a new one. Make sure the certificate is created in the same region as your cluster.
Note the certificate ARN, you will need it later.

### Create NodePort services for the console

The AWS Load Balancer Controller routes the traffic between cluster instances.
Therefore the service needs to be exposed via a port on the instance network:
```bash
cat <<EOF | oc apply -f -
apiVersion: v1
kind: Service
metadata:
  labels:
    app: console
  name: console-np
  namespace: openshift-console
spec:
  ports:
  - name: https
    port: 443
    protocol: TCP
    targetPort: 8443
  selector:
    app: console
    component: ui
  type: NodePort
EOF
```

### Create Ingress for the console service

Create an Ingress resource to provision an ALB:
```bash
cat <<EOF | oc apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: instance
    alb.ingress.kubernetes.io/backend-protocol: HTTPS
    alb.ingress.kubernetes.io/certificate-arn: ${CERTIFICATE_ARN}
  name: console
  namespace: openshift-console
spec:
  ingressClassName: alb
  rules:
    - http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: console-np
                port:
                  number: 443
EOF
```

### Update console config

The OpenShift console checks for the origin URI of the incoming requests. It has to match the console base address from the console configuration.
Otherwise the following error is produced:

> middleware.go:60] invalid source origin: invalid Origin or Referer: https://k8s-openshif-console-xxxxxxxxxx-xxxxxxxxx.us-east-2.elb.amazonaws.com expected `https://console-openshift-console.apps.mytestcluster.devcluster.openshift.com/

Edit `console-config` configmap by replacing `clusterInfo.consoleBaseAddress` field with the hostname of the provisioned ALB:
```bash
$ CONSOLE_ALB_HOST=$(oc -n openshift-console get ing console -o yaml | yq .status.loadBalancer.ingress[0].hostname)
$ oc -n openshift-console edit cm console-config
```

**Gaps**
- The console operator reconciles the console configuration, overriding any changes.
- The console pods need to be recreated to read the new config.

### Update console OAuthClient

The console uses a dedicated oauthclient to set the redirect link. We need to change it so that the authentication server redirects back to the provisioned ALB:
```bash
$ oc patch oauthclient console --type='json' -p="[{\"op\": \"replace\", \"path\": \"/redirectURIs/0\", \"value\":\"https://${CONSOLE_ALB_HOST}/auth/callback\"}]"
```

**Gaps**
- The console operator reconciles the console OAuthClient, overriding any changes.

## Notes

1. ROSA HCP does not have the authentication operator, the authentication server is managed centrally by the HyperShift layer:
```bash
$ oc -n openshift-authentication-operator get deploy
No resources found in openshift-authentication-operator namespace.

$ oc -n openshift-authentication-operator get route
No resources found in openshift-authentication-operator namespace.

$ oc -n openshift-authentication get pods
No resources found in openshift-authentication namespace.

$ oc -n openshift-authentication get routes
No resources found in openshift-authentication namespace.

$ oc get oauthclient | grep -v console
NAME                           SECRET                                        WWW-CHALLENGE   TOKEN-MAX-AGE   REDIRECT URIS
openshift-browser-client                                                     false           default         https://oauth.mytestcluster.5199.s3.devshift.org:443/oauth/token/display
openshift-challenging-client                                                 true            default         https://oauth.mytestcluster.5199.s3.devshift.org:443/oauth/token/implicit

$ oc -n openshift-console rsh deploy/console curl -k https://openshift.default.svc/.well-known/oauth-authorization-server
{
"issuer": "https://oauth.mytestcluster.5199.s3.devshift.org:443",
"authorization_endpoint": "https://oauth.mytestcluster.5199.s3.devshift.org:443/oauth/authorize",
"token_endpoint": "https://oauth.mytestcluster.5199.s3.devshift.org:443/oauth/token",
```

2. To simulate a disabled ingress set the desired replicas to zero in the default ingress controller:
```bash
$ oc -n openshift-ingress-operator patch ingresscontroller default --type='json' -p='[{"op": "replace", "path": "/spec/replicas", "value":0}]'
```

## Links
- [Small demo of ALB ingress for the console on ROSA HCP](https://drive.google.com/file/d/1uWZgFbSeZTlDzlFyPW7QcH-625JsbSbw/view)

# Deploying on Kubernetes

## Testing

To test csi-cloudscale in conjunction with Kubernetes, a suite of integration tests has been implemented.
To run this test suite, a Kubernetes cluster is required. For this purpose, this setup was prepared using
[k8test](https://github.com/cloudscale-ch/k8test).

> ⚠️ Running these tests yourself may incur unexpected costs and may result in data loss if run against a production account with live systems. herefore, we strongly advise you to use a separate account for these tests.
> The Kubernetes cluster created is not production ready and should not be used for any purpose other than testing.

First bootstrap the cluster

    # Export your API Token obtained from http://control.cloudscale.ch
    export CLOUDSCALE_API_TOKEN="..."
    
    # See the script for options, sensible defaults apply
    ./helpers/bootstrap-cluster
    
    # Verify cluster setup and access
    export KUBECONFIG=$PWD/k8test/cluster/admin.conf
    kubectl get nodes -o wide


You can **either** install the driver from your working directory

    # Install driver using dev image from working dir
    # Pre-requesit: ensure the you have run `helm dependency build` as described in the main README file.
    helm install -g -n kube-system --set controller.image.tag=dev --set node.image.tag=dev --set controller.image.pullPolicy=Always --set node.image.pullPolicy=Always ./charts/csi-cloudscale

**Or** you can install a released version:

    # List all released versions
    helm search repo csi-cloudscale/csi-cloudscale  --versions
    # Install a specific Chart version or latest if --version is omitted
    helm install -n kube-system -g csi-cloudscale/csi-cloudscale [ --version v1.0.0 ]

Then execute the test suite:

    make test-integration

The get rid of the cluster:

    ./helpers/clean-up

## Debugging

If the suite does not pass, there are a good number of ways to debug.
You can just redeploy all csi pods and push a new version to docker hub:

    VERSION=dev make publish
    kubectl -n kube-system delete pods csi-cloudscale-controller-0 csi-cloudscale-node-pxswm csi-cloudscale-node-skgw4

Kubernetes will then automatically reinstall the pods you just deleted
with newer versions.

To follow logs on Kubernetes master server:

    kubectl logs -n kube-system csi-cloudscale-controller-0 --all-containers --timestamps -f

Some commands:

    kubectl logs -n kube-system -l role=csi-cloudscale --all-containers --timestamps


    kubectl logs -n kube-system csi-cloudscale-controller-0 -c csi-cloudscale-plugin

    kubectl logs -n kube-system -l role=csi-cloudscale -c csi-cloudscale-plugin
    kubectl logs -n kube-system -l role=csi-cloudscale -c csi-provisioner
    kubectl logs -n kube-system -l role=csi-cloudscale -c csi-attacher

    kubectl logs -n kube-system -c driver-registrar csi-cloudscale-node-7s585
    kubectl logs -n kube-system kube-dns-56df66c9f-hgjm8 --all-containers --timestamps

    kubectl get pods -n kube-system
    kubectl get namespaces

List all containers:

    kubectl get pods --namespace kube-system -o jsonpath="{..image}" | xargs -n 1 echo
    kubectl get pods -n=kube-system -o jsonpath={.items[*].spec.containers[*].name} | xargs -n 1 echo

To attach:

    docker exec -it k8s_csi-cloudscale-plugin_csi-cloudscale-node-qpkube-system_3e87d88b-cc98-11e8-9d29-5a4205669245_3 /bin/sh

Granting admin privileges to Dashboard\'s Service Account:

    $ cat <<EOF | kubectl create -f -
    apiVersion: rbac.authorization.k8s.io/v1beta1
    kind: ClusterRoleBinding
    metadata:
      name: kubernetes-dashboard
      labels:
        k8s-app: kubernetes-dashboard
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: cluster-admin
    subjects:
    - kind: ServiceAccount
      name: kubernetes-dashboard
      namespace: kube-system
    EOF

Access the dashboard:

    https://<IP>:6443/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/

Using etcdctl:

    ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379  --cert /etc/calico/certs/ca_cert.crt --cert /etc/calico/certs/cert.crt --key /etc/calico/certs/key.pem endpoint health

    etcdctl ... get  / --prefix --keys-only

\# Volume status

> grep \'server:\' \~/.kube/config \# get cluster from string:
> <https://1.1.1.1/k8s/clusters/c-xfmg6> kubectl get nodes \# get nodes
> name kubectl get \--raw /k8s/clusters/{}/api/v1/nodes/{}/proxy/metrics
> \| grep kubelet_vol

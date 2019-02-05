Deploying on Kubernetes
=======================

Testing
-------

To test kubernetes csi-cloudscale on kubernetes, you can install kubespray to
deploy kubernetes::

    python3 -m venv venv
    . venv/bin/activate
    pip install -r kubespray/requirements.txt

After this you run::

    CLOUDSCALE_TOKEN="foobar" ansible-playbook integration_test.yml -i inventory/hosts.ini

to install kubernetes on cloudscale.ch and run the integration tests. The
playbook will also clean up VMs after the test.

If you want to test a fresh release, you can use an additional ``-e version=v1.0.0``.

Debugging
---------

If the playbook does not pass, there are a good number of ways to debug. You
can just redeploy all csi pods and push a new version to docker hub::

    VERSION=dev make publish
    kubectl -n kube-system delete pods csi-cloudscale-controller-0 csi-cloudscale-node-pxswm csi-cloudscale-node-skgw4

Kubernetes will then automatically reinstall the pods you just deleted with
newer versions.

To follow logs on kubernetes master server::

    kubectl logs -n kube-system csi-cloudscale-controller-0 --all-containers --timestamps -f

Some commands::

    kubectl logs -n kube-system -l role=csi-cloudscale --all-containers --timestamps


    kubectl logs -n kube-system csi-cloudscale-controller-0 -c csi-cloudscale-plugin

    kubectl logs -n kube-system -l role=csi-cloudscale -c csi-cloudscale-plugin 
    kubectl logs -n kube-system -l role=csi-cloudscale -c csi-provisioner
    kubectl logs -n kube-system -l role=csi-cloudscale -c csi-attacher

    kubectl logs -n kube-system -c driver-registrar csi-cloudscale-node-7s585
    kubectl logs -n kube-system kube-dns-56df66c9f-hgjm8 --all-containers --timestamps

    kubectl get pods -n kube-system 
    kubectl get namespaces

List all containers::

    kubectl get pods --namespace kube-system -o jsonpath="{..image}" | xargs -n 1 echo
    kubectl get pods -n=kube-system -o jsonpath={.items[*].spec.containers[*].name} | xargs -n 1 echo


To attach::

    docker exec -it k8s_csi-cloudscale-plugin_csi-cloudscale-node-qpkube-system_3e87d88b-cc98-11e8-9d29-5a4205669245_3 /bin/sh

Granting admin privileges to Dashboard's Service Account::

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

Access the dashboard::

    https://<IP>:6443/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/


Using etcdctl::

    ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379  --cert /etc/calico/certs/ca_cert.crt --cert /etc/calico/certs/cert.crt --key /etc/calico/certs/key.pem endpoint health

    etcdctl ... get  / --prefix --keys-only


Ansible::

    ansible-playbook -i inventory/hosts.ini integration_test.yml --skip-tags install-kubernetes --skip-tags cleanup

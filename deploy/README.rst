Deploying on Kubernetes
=======================

Testing
-------

To test kubernetes csi-cloudscale on kubernetes, you can install kubespray to
deploy kubernetes::

    python3 -m venv venv
    . venv/bin/activate
    # or requirements-{VERSION}.txt, see https://github.com/kubernetes-sigs/kubespray/blob/master/docs/ansible.md#ansible-python-compatibility
    pip install -r kubespray/requirements.txt
    cd kubespray/

After this you run::

    CLOUDSCALE_TOKEN="foobar" ansible-playbook ../integration_test.yml -i inventory/hosts.ini

to install kubernetes on cloudscale.ch and run the integration tests. The
playbook will also clean up VMs after the test.

* If you just want to provision a cluster, you can use an additional  ``--skip-tags cleanup --skip-tags test``.

* If you want to a test release other than ``dev``, you can use an additional ``-e version=v1.0.0``.

* If you want to use a non-default Kubernetes version, you can use an additional ``-e kube_version=v1.20.7``.

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

# Volume status

    grep 'server:' ~/.kube/config   # get cluster from string: https://1.1.1.1/k8s/clusters/c-xfmg6
    kubectl get nodes               # get nodes name
    kubectl get --raw /k8s/clusters/{}/api/v1/nodes/{}/proxy/metrics  | grep kubelet_vol


Ansible::

    # Keep cluster after test run
    CLOUDSCALE_TOKEN="foobar" ansible-playbook integration_test.yml -i inventory/hosts.ini --skip-tags cleanup

    # Just run tests
    ansible-playbook -i inventory/hosts.ini integration_test.yml --tags test

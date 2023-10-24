.. 
   Copyright (C) 2023, Advanced Micro Devices, Inc. - All rights reserved
  
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
  
       http://www.apache.org/licenses/LICENSE-2.0
  
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

.. _install.rst:

Installing FPGA Operator
------------------------

1. Installing Using Helm
^^^^^^^^^^^^^^^^^^^^^^^^

Helm is a tool used to manage Kubernetes applications, and Helm chart can define, install, and upgrade even the most complex Kubernetes application. 
It is the preferred way to install the FPGA-Operator.

``Note``: If you have previously installed the old version of FPGA-Operator, the helm will not be able to update the CRD by 'install' command.
Therefore, you need to update the CRD manually via `kubectl <https://kubernetes.io/docs/reference/kubectl/>`_.

.. code-block:: bash

    $ kubectl apply -f https://raw.githubusercontent.com/Xilinx/fpga-operator/main/deployments/fpga-operator/crds/policy.xilinx.com_clusterpolicies.yaml


1.1 Helm Repo
.............

First, add the Xilinx Helm repository which contains fpga-operator Helm chart:

.. code-block:: bash
    
    $ helm repo add xilinx https://xilinx.github.io/fpga-operator && helm repo update


1.2 Namespace
.............

The FPGA-Operator will be installed in the default namespace if not specified, and it is configurable and determined during installation. 
For example, to install the FPGA-Operator in 'xilinx-system' namespace:

.. code-block:: bash
    
    $ helm install --generate-name -n xilinx-system --create-namespace xilinx/fpga-operator


2. Installing From Source Code
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Other than installing from Helm repo, the FPGA-Operator can be deployed from source code, which contains all the Kubernetes resources definition for FPGA-Operator.

2.1 Deploying NFD
.................

Before the deployment of FPGA-Operator, we need NFD to create few labels on each node in the cluster.

.. code-block:: bash

    $ kubectl apply -k https://github.com/kubernetes-sigs/node-feature-discovery/deployment/overlays/default?ref=v0.12.1

2.2 Getting Source Code
.......................

.. code-block:: bash

    $ git clone https://github.com/Xilinx/fpga-operator.git && cd ./fpga-operator

2.3 Deploying CRD and Operator
..............................

This step will create CustomResourceDefinition(ClusterPolicy) in the cluster specified in ``~/.kube/config`` and deploy operator controller. 
By default, all the resources will be created in namespace "xilinx-system". To change the namespace, update ``./config/default/kustomization.yaml`` properly.

.. code-block:: bash

    $ make deploy

2.4 Creating ClusterPolicy Object
.................................

This step will create a ClusterPolicy object and trigger the operator controller to create the required Kubernetes resources.

.. code-block:: bash

    $ kubectl apply -f ./config/samples/policy_v1_clusterpolicy.yaml

3. Expected Result
^^^^^^^^^^^^^^^^^^

FPGA-Operator will create few Kubernetes resources in the specified cluster. 
Once all the setup steps are finished, the pods similar to the following will be created and running.

.. code-block:: bash

    $ kubectl get pod -n xilinx-system
    NAME                                                              READY   STATUS    RESTARTS        AGE
    device-plugin-daemonset-pj8z2                                     1/1     Running   1               3m1s
    fpga-operator-1675810317-node-feature-discovery-master-867lxdsr   1/1     Running   0               3m46s
    fpga-operator-1675810317-node-feature-discovery-worker-m8dlp      1/1     Running   1 (3m23s ago)   3m46s
    fpga-operator-54d888ffcc-rg86m                                    1/1     Running   0               3m46s
    host-setup-ubuntu18-daemonset-bmd98                               1/1     Running   0               3m1s
    xilinx-container-runtime-daemonset-s7g9k                          1/1     Running   0               3m1s


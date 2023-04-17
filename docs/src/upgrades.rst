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

.. _upgrades.rst:

Upgrade and Uninstallation
--------------------------

No matter you make the deployment using Helm chart or from source code, it is always easy to update the existing deployment.

Helm Chart
^^^^^^^^^^

After the FPGA-Operator chart is installed, you should see the chart details as following:

.. code-block:: bash

    $ helm install --generate-name -n xilinx-system --create-namespace xilinx/fpga-operator
    NAME: fpga-operator-1675794640
    LAST DEPLOYED: Tue Feb  7 10:30:41 2023
    NAMESPACE: xilinx-system
    STATUS: deployed
    REVISION: 1
    TEST SUITE: None

    $ helm list --all-namespaces
    NAME                            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART                   APP VERSION
    fpga-operator-1675794640        xilinx-system   1               2023-02-07 10:30:41.043301696 -0800 PST deployed        fpga-operator-0.1.0     0.1.0-alpha


Upgrade
.......

Helm provides 'upgrade' command to update a chart to a new revision.
To make the update, we need to specify the chart release name and chart path. The namespace should be specified with '-n' unless it was deployed in the default namespace. 
Refer to `Helm Upgrade <https://helm.sh/docs/helm/helm_upgrade/>`_ for more details.

The following example is trying to update the chart release with 'nfd.enabled=false'. 

.. code-block:: bash

    $ helm upgrade fpga-operator-1675794640 myxilinx/fpga-operator -n xilinx-system --set nfd.enabled=false
    Release "fpga-operator-1675794640" has been upgraded. Happy Helming!
    NAME: fpga-operator-1675794640
    LAST DEPLOYED: Tue Feb  7 11:13:53 2023
    NAMESPACE: xilinx-system
    STATUS: deployed
    REVISION: 2
    TEST SUITE: None
  
Uninstall
.........

It is easy to uninstall the Helm chart using 'uninstall' command. 
However, Xilinx_Container_Runtime and XRT are both installed on host, so they will be kept on host even the chart is uninstalled.

.. code-block:: bash

    $ helm uninstall fpga-operator-1675794640 -n xilinx-system
    release "fpga-operator-1675794640" uninstalled


Source Code
^^^^^^^^^^^

Upgrade
.......

The update can be performed by updating the ClusterPolicy object, defined by file ``./config/samples/policy_v1_clusterpolicy.yaml``.
After the edit of this file, simply run ``kubectl apply`` to update the ClusterPolicy object.

.. code-block:: bash

    $ kubectl apply -f ./config/samples/policy_v1_clusterpolicy.yaml
    clusterpolicy.policy.xilinx.com/fpga-clusterpolicy configured

The change of ClusterPolicy object will trigger the reconcile loop of FPGA-Operator controller, leading to the Kubernetes resources update if needed.

Uninstall
.........

To uninstall FPGA-Operator from source code, we need to delete ClusterPolicy object first, then the operator controller and CRD.

.. code-block:: bash

    $ kubectl delete -f ./config/samples/policy_v1_clusterpolicy.yaml
    clusterpolicy.policy.xilinx.com "fpga-clusterpolicy" deleted

    $ make undeploy
    namespace "xilinx-system" deleted
    customresourcedefinition.apiextensions.k8s.io "clusterpolicies.policy.xilinx.com" deleted
    serviceaccount "fpga-operator" deleted
    role.rbac.authorization.k8s.io "leader-election-role" deleted
    clusterrole.rbac.authorization.k8s.io "fpga-operator-role" deleted
    clusterrole.rbac.authorization.k8s.io "metrics-reader" deleted
    clusterrole.rbac.authorization.k8s.io "proxy-role" deleted
    rolebinding.rbac.authorization.k8s.io "leader-election-rolebinding" deleted
    clusterrolebinding.rbac.authorization.k8s.io "fpga-operator-rolebinding" deleted
    clusterrolebinding.rbac.authorization.k8s.io "proxy-rolebinding" deleted
    configmap "manager-config" deleted
    service "fpga-operator-metrics-service" deleted
    deployment.apps "fpga-operator" deleted


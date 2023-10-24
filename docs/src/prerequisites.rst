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

.. _prerequisites.rst:

Prerequisites
-------------

Kubernetes
~~~~~~~~~~

Refer to `kubernetes.io <https://kubernetes.io/docs/setup/>`_ for getting started with setting up a Kubernetes cluster.

Before installing the FPGA-Operator, you should ensure that the Kubernetes cluster meets some prerequisites.

#. Nodes must be configured with a container engine such as Docker or containerd.
#. Node Feature Discovery (NFD) is used to create labels on each node. By default, NFD master and worker are automatically deployed by the operator. If NFD is already running in the cluster prior to the deployment of the operator, the operator can be configured to not to install NFD.


Helm
~~~~
Helm is a tool that streamlines installing and managing Kubernetes applications, and it is the preferred method to deploy the FPGA-Operator.
Helm can be installed from the script below.

.. code-block:: bash

    $ curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
    $ chmod 700 get_helm.sh
    $ ./get_helm.sh


Refer to `Install Helm <https://helm.sh/docs/intro/install/>`_ for more details.

OS Distributions
~~~~~~~~~~~~~~~~
FPGA Opeartor has been tested on the following OS distributions:

#. Ubuntu 18.04
#. Ubuntu 20.04
#. Ubuntu 22.04
#. CentOS 7

FGPA Devices
~~~~~~~~~~~~
Xilinx Container Runtime and Device Plugin can work with all Xilinx devices.
However, the FPGA Operator is only able to install XRT and flash shells for the following Xilinx alveo cards:

#. Alveo U200
#. Alveo U250
#. Alveo U280
#. Alveo U50
#. Alveo U55c

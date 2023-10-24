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

.. _customization.rst:

Deployment Customization
------------------------

For the deployment of FPGA-Operator, there are many customizable options. 
Helm allows users to customize the chart using a customized values file or ``--set`` flag. 

``Note``: If you are making the deployment from the source code, the file `policy_v1_clusterpolicy.yaml <https://github.com/Xilinx/fpga-operator/blob/main/config/samples/policy_v1_clusterpolicy.yaml>`_ can be edited to customize the ClusterPolicy object.

Helm Chart
^^^^^^^^^^

Values File
...........

The FPGA-Operator Helm chart offers several customizable options that can be configured depending on your request.
Helm chart uses `values.yaml <https://github.com/Xilinx/fpga-operator/blob/main/deployments/fpga-operator/values.yaml>`_ as the default configuration, and you can use your own values file for the chart installation.

.. code-block:: bash

    $ vi ./my-values.yaml
    nfd:
      enabled: false
    .........
    $ helm install --generate-name -n xilinx-system --create-namespace xilinx/fpga-operator -f ./my-values.yaml

Command Line Flag
.................

If you want to use the default values file and customize only a few of the options, ``--set`` flag is an easy way to make it.
The following table lists part of the available options of the chart, and the full list can be found from `values.yaml <https://github.com/Xilinx/fpga-opeartor/blob/main/deployments/fpga-operator/values.yaml>`_.

.. list-table:: FPGA-Operator Options
   :widths: 20 65 15
   :header-rows: 1

   * - Parameter
     - Description
     - Default
   * - ``nfd.enabled``
     - | Deploys Node Feature Discovery plugin as a daemonset.
       | Set this variable to false if NFD is already running in the cluster.
     - ``true``
   * - ``operator.defaultRuntime``
     - | FPGA-Operator will dectect the default CRI used by the Kubernetes cluster automatically.
       | This value will be used in case the CRI is not detected.
       | Allowed values: ``docker/containerd``
     - ``containerd``
   * - ``containerRuntime.enabled``
     - | Installs xilinx-container-runtime and create a runtimeclass.
       | Set this variable to false if xilinx-container-runtime is installed already or not needed.
     - ``true``
   * - ``devicePlugin.enabled``
     - Deploys a device-plugin daemonset to create allocatable Kubernetes device resources.
     - ``true``
   * - ``hostSetup.enabled``
     - | Installs XRT and flash cards. 
       | Set this variable to false if XRT has been installed and the cards have been flashed already. 
     - ``true``

Here is an example to install FPGA Opeartor with NFD disabled.

.. code-block:: bash
    
    $ helm install --generate-name -n xilinx-system --create-namespace xilinx/fpga-operator --set nfd.enabled=false


Host Setup
^^^^^^^^^^

Host Setup is used to install XRT on the host and flash Xilinx Alveo cards with specified versions. 
However, not all the Aleveo cards and Linux distributions are supported. Visit `Xilinx Runtime Base <https://github.com/Xilinx/Xilinx_Base_Runtime#available-docker-images>`_ to view the full support list. 
If this is enabled with unsupported cards or Linux distributions, nothing will be performed.

In FPGA-Operator, HostSetup can be customized per Linux distribution. Here is an example of configuration in the values file for Ubuntu 18 and Ubuntu 20.

.. code-block:: yaml
    
    hostSetup:
      # install xrt and shell; flash cards
      enabled: true
      osDists:
      - osId: ubuntu
        osMajorVersion: "18"
        version: "2023.1"
        xrmInstallation: true
        shellFlashEnabled: true # default value is true
        cards: [] # empty to perform setup for all cards; 
        # cards: ["alveo-u200", "alveo-u50"]
        repository: public.ecr.aws/xilinx_dcg
        image: host-setup
        tag: ubuntu18.04
        imagePullPolicy: IfNotPresent
      - osId: ubuntu
        osMajorVersion: "20"
        version: "2023.1"
        xrmInstallation: true
        shellFlashEnabled: true # default value is true
        cards: [] # empty to perform setup for all cards
        repository: public.ecr.aws/xilinx_dcg
        image: host-setup
        tag: ubuntu20.04
        imagePullPolicy: IfNotPresent

To set the values using ``--set`` and ``--set-string`` flag:

.. code-block:: bash
    
    $ helm install --generate-name -n xilinx-system --create-namespace xilinx/fpga-operator \
        --set hostSetup.osDists[0].osId=ubuntu \
        --set-string hostSetup.osDists[0].osMajorVersion=18 \
        --set-string hostSetup.osDists[0].version=2022.1 \
        --set hostSetup.osDists[0].repository=public.ecr.aws/xilinx_dcg \
        --set hostSetup.osDists[0].image=host-setup \
        --set hostSetup.osDists[0].tag=ubuntu18.04 \
        --set hostSetup.osDists[1].osId=ubuntu \
        --set-string hostSetup.osDists[1].osMajorVersion=20 \
        --set-string hostSetup.osDists[1].version=2022.1 \
        --set hostSetup.osDists[1].repository=public.ecr.aws/xilinx_dcg \
        --set hostSetup.osDists[1].image=host-setup \
        --set hostSetup.osDists[1].tag=ubuntu20.04


Device Plugin
^^^^^^^^^^^^^

The FPGA Device Plugin is used to advertise Xilinx FPGA devices to the Kubelet, and it can be customized via environment variables.

For example, each Xilinx U30 card has two character devices, and you can choose to allocate a U30 computing unit based on either card or device.
We can simply add the environment variable 'U30AllocUnit' and set it as 'Card' or 'Device' in the values file.

.. code-block:: yaml
    
    devicePlugin:
      # deploy a device-plugin daemonset
      enabled: true
      repository: public.ecr.aws/xilinx_dcg
      image: k8s-device-plugin
      tag: 1.2.0
      imagePullPolicy: IfNotPresent
      env:
      - name: U30NameConvention
        value: CommonName
      - name: U30AllocUnit
        value: Card

To set values using ``--set`` flag:

.. code-block:: bash
    
    $ helm install --generate-name -n xilinx-system --create-namespace xilinx/fpga-operator \
        --set devicePlugin.repository=public.ecr.aws/xilinx_dcg \
        --set devicePlugin.image=k8s-device-plugin \
        --set devicePlugin.tag=1.2.0 \
        --set devicePlugin.env[0].name=U30NameConvention \
        --set devicePlugin.env[0].value=CommonName \
        --set devicePlugin.env[1].name=U30AllocUnit \
        --set devicePlugin.env[1].value=Card


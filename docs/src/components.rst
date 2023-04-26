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

.. _components.rst:

Major Components
----------------

The FPGA Opeartor is able to automate the deployments of three components to provision FPGA devices in a Kubernetes cluster, which will be detailed below.
Also, a third party component "node-feature-discovery" is used to label nodes, for FPGA-Operator to get some node info.

Node Feature Discovery
^^^^^^^^^^^^^^^^^^^^^^^

Node Feature Fiscovery (NFD) is a Kubernetes add-on for detecting hardware features and system configuration. FPGA-Operator will use these labels to detect whether FPGA devices are installed on node and node OS info. 
FPGA-Operator will deploy NFD master and worker automatically, unless flag 'nfd.enabled' is set to false.

Once deployed, some lables with prefix 'feature.node.kubernetes.io' will be created.
Refer to `Node Feature Discovery <https://kubernetes-sigs.github.io/node-feature-discovery/stable/get-started/index.html>`_ for more details of Node Feature Discovery. 

.. code-block:: bash
    
    $ kubectl describe node | grep feature.node.kubernetes.io
                    feature.node.kubernetes.io/cpu-cpuid.AESNI=true
                    feature.node.kubernetes.io/cpu-cpuid.AVX=true
                    feature.node.kubernetes.io/cpu-cpuid.AVX2=true
                    feature.node.kubernetes.io/cpu-cpuid.CMPXCHG8=true
                    feature.node.kubernetes.io/cpu-cpuid.FLUSH_L1D=true
                    feature.node.kubernetes.io/cpu-cpuid.FMA3=true
                    feature.node.kubernetes.io/cpu-cpuid.FXSR=true
                    feature.node.kubernetes.io/cpu-cpuid.FXSROPT=true
                    feature.node.kubernetes.io/cpu-cpuid.IBPB=true
                    feature.node.kubernetes.io/cpu-cpuid.LAHF=true
    .......

Host Setup
^^^^^^^^^^^

For running a container of FPGA applications, the FPGA devices must be flashed with shells, and Xilinx Runtime (XRT) is required to install on host. This component will automate the XRT installation and shell flashing processes.

This step only supports Xilinx Aleveo FPGA devices. Refer to `Xilinx Base Runtime <https://github.com/Xilinx/Xilinx_Base_Runtime>`_ to get more infomation and a full list of supported platforms. 

In FPGA-Operator, we have the host-setup pod with two init-containers to perform XRT installation and card flashing.
During the process of setup, the pod status is something like ``Init:1/2``. 
The process may take long depending on the cards on the host, and the pod status will become ``Running`` once it is done. 

After the XRT installation and card flashing process are finished, a cold-reboot is required.

.. code-block:: bash
    
    $ kubectl get pod -n xilinx-system --watch
    NAME                                                              READY   STATUS              RESTARTS   AGE
    ......
    host-setup-ubuntu18-daemonset-kw2zk                               0/1     Init:1/2            0          9s
    host-setup-ubuntu18-daemonset-kw2zk                               0/1     Init:2/2            0          45s
    host-setup-ubuntu18-daemonset-kw2zk                               1/1     Running             0          1m


Xilinx Container Runtime
^^^^^^^^^^^^^^^^^^^^^^^^

Xilinx container runtime is an extension of runC, with modification to add xilinx devices before running containers. Since it is a runC compliant runtime, xilinx container runtime can be integrated with various contianer orchestrators, including docker and podman.
Refer to `Xilinx Container Runtime <https://xilinx.github.io/Xilinx_Container_Runtime/>`_ for more details.

FPGA-Operator will install Xilinx container runtime on each node, and modify the containerd configuration to add a handler leveraging Xilinx container runtime.
In addition, a `RuntimeClass <https://kubernetes.io/docs/concepts/containers/runtime-class/>`_ referring to Xilinx container runtime will be created.

.. code-block:: bash
    
    $ kubectl get runtimeclass
    NAME     HANDLER   AGE
    xilinx   xilinx    48m


Device Plugin
^^^^^^^^^^^^^^

Kubernetes provides a device plugin framework that can be used to advertise system hardware resources to the Kubelet. 
We have a FPGA device plugin to expose Xilinx FPGA devices to Kubernetes cluster.

Once the FPGA device plugin is deployed, we can check the device resources on each node and find Xilixn FPGA devices.
Refer to `FPGA Device Plugin <https://github.com/Xilinx/FPGA_as_a_Service/tree/master/k8s-device-plugin>`_ for more details.

.. code-block:: bash
    
    $ kubectl describe node
    ......
    Capacity:
        amd.com/xilinx_u200_gen3x16_xdma_base_2-0:     1
        amd.com/xilinx_u200_qdma_201920_1-1573695690:  0
        amd.com/xilinx_u50_gen3x16_xdma_201920_3-0:    0
        amd.com/xilinx_u50_gen3x16_xdma_base_5-0:      1
        cpu:                                           12
        ephemeral-storage:                             362372628Ki
        hugepages-1Gi:                                 0
        hugepages-2Mi:                                 0
        memory:                                        98904344Ki
        pods:                                          110
    ......

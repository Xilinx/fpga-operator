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

.. _samples.rst:

Running Sample FPGA Application
-------------------------------

Once the whole deployment is done, the FPGA devices can be assigned to a pod.
In the following example, we will create a pod with one FPGA device.

Checking Available Devices
..........................

In the following example, we have one ``amd.com/xilinx_u200_gen3x16_xdma_base_2-0`` available in the cluster, and we will use this device in the pod to be created.

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

Creating Pod
............

We have `Docker images <https://hub.docker.com/repository/docker/xilinx/xilinx_runtime_base/general>`_ ready with XRT installed, which can be used as base images to run FPGA applications.
Here is an example of creating the pod with XRT Docker image.

.. code-block:: bash
    
    $ cat << EOF | kubectl create -f -
    apiVersion: v1
    kind: Pod
    metadata:
      name: xrt-sample
    spec:
      containers:
      - name: xrt-sample
        image: xilinx/xilinx_runtime_base:alveo-2022.2-ubuntu-18.04
        imagePullPolicy: IfNotPresent
        command: ["/bin/sh"]
        args: ["-c", "while true; do echo hello; sleep 3600; done"]
        resources:
          limits:
            amd.com/xilinx_u200_gen3x16_xdma_base_2-0: 1
    EOF


SSH into Container
..................

Once the pod is created and running, we can ssh into the container and check whether the FPGA device is mounted into the container properly.

.. code-block:: bash
    
    # the pod is running up
    $ kubectl get pod
    NAME         READY   STATUS    RESTARTS   AGE
    xrt-sample   1/1     Running   0          109s

    # ssh into pod
    $ kubectl exec -it xrt-sample -- /bin/bash

    # running in pod
    $ source /opt/xilinx/xrt/setup.sh
    $ xbutil examine
    WARNING: Unexpected xocl version (2.13.466) was found. Expected 2.14.354, to match XRT tools.
    System Configuration
    OS Name              : Linux
    Release              : 4.15.0-23-generic
    Version              : #25-Ubuntu SMP Wed May 23 18:02:16 UTC 2018
    Machine              : x86_64
    CPU Cores            : 12
    Memory               : 96586 MB
    Distribution         : Ubuntu 18.04.6 LTS
    GLIBC                : 2.27
    Model                : PowerEdge R730

    XRT
    Version              : 2.14.354
    Branch               : 2022.2
    Hash                 : 43926231f7183688add2dccfd391b36a1f000bea
    Hash Date            : 2022-10-08 09:51:53
    XOCL                 : 2.13.466, f5505e402c2ca1ffe45eb6d3a9399b23a0dc8776
    XCLMGMT              : 2.13.466, f5505e402c2ca1ffe45eb6d3a9399b23a0dc8776

    Devices present
    BDF             :  Shell                            Platform UUID                         Device ID       Device Ready*
    -------------------------------------------------------------------------------------------------------------------------
    [0000:03:00.1]  :  xilinx_u200_gen3x16_xdma_base_2  0B095B81-FA2B-E6BD-4524-72B1C1474F18  user(inst=128)  Yes


    * Devices that are not ready will have reduced functionality when using XRT tools


The above result shows that the device is ready to be used in the container, and we are able to run the application with this device.

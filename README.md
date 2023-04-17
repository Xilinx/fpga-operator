<!--
 Copyright (C) 2023, Advanced Micro Devices, Inc. - All rights reserved
 FPGA Operator
 
 Licensed under the Apache License, Version 2.0 (the "License"). You may
 not use this file except in compliance with the License. A copy of the
 License is located at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 License for the specific language governing permissions and limitations
 under the License. 
-->

# FPGA Operator

<image src="./docs/_static/fpga-operator.png" width=50%>

<br>

Kubernetes provides access to special hardware resources such as Xilinx FPGAs and other devices through the device plugin framework. However, configuring and managing nodes with these hardwares requires configuration of multiple software components such as XRT, container runtime and device plugin. The FPGA Operator uses the operator framework within Kubernetes to automate the management of Xilinx software components needed to provision Xilinx FPGA devices.

## Getting Started

### [Prerequisites](./docs/prerequisites.rst)

### [Install FPGA Operator](./docs/install.rst)

<br>

## User Guide

### [Major Components](./docs/components.rst)

### [Deployment Customization](./docs/customization.rst)

### [Upgrade and Uninstallation](./docs/upgrades.rst)

### [Running Sample FPGA Application](./docs/samples.rst)

<br>

## Contact
 
### k8s_dev@amd.com

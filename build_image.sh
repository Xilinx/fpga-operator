#
# Copyright (C) 2023, Advanced Micro Devices, Inc. - All rights reserved
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#!/usr/bin/env bash

YES=0
TARGET=""
IMAGE_REPO=""
IMAGE_VERSION=""

confirm() {
    read -r -p "${1:-CHECK: Do you want to proceed? [y/n]:} " response
    case "$response" in
        [yY][eE][sS]|[yY])
            true
        ;;
        *)
            exit 1
        ;;
    esac
}

while true
do
    case "$1" in
        -t|--target         ) TARGET="$2"       ; shift 2   ;;
        -r|--image_repo     ) IMAGE_REPO="$2"   ; shift 2   ;;
        -v|--image_version  ) IMAGE_VERSION="$2"; shift 2   ;;
        -y|--yes            ) YES=1             ; shift 1   ;;
        ""                  ) break             ;;
        *) echo "ERROR: Invalid option: $1"; exit 1 ;;
    esac
done

TAEGET="${TARGET,,}"
IMAGE="host-setup"
case "$TAEGET" in
    centos7)
        cat ./notices/NOTICE_centos7.txt
        if [[ "$YES" != 1 ]]; then
            confirm
        fi
        if [[ "$IMAGE_REPO" != "" ]]; then
            IMAGE="$IMAGE_REPO/$IMAGE"
        fi
        if [[ "$IMAGE_VERSION" == "" ]]; then
            IMAGE_VERSION="centos7.9"
        fi
        docker build -t ${IMAGE}:${IMAGE_VERSION} -f ./hostSetup/Dockerfile.centos7 ./hostSetup
    ;;
    ubuntu18)
        cat ./notices/NOTICE_ubuntu18.txt
        if [[ "$YES" != 1 ]]; then
            confirm
        fi
        if [[ "$IMAGE_REPO" != "" ]]; then
            IMAGE="$IMAGE_REPO/$IMAGE"
        fi
        if [[ "$IMAGE_VERSION" == "" ]]; then
            IMAGE_VERSION="ubuntu18.04"
        fi
        docker build -t ${IMAGE}:${IMAGE_VERSION} -f ./hostSetup/Dockerfile.ubuntu18 ./hostSetup
    ;;
    ubuntu20)
        cat ./notices/NOTICE_ubuntu20.txt
        if [[ "$YES" != 1 ]]; then
            confirm
        fi
        if [[ "$IMAGE_REPO" != "" ]]; then
            IMAGE="$IMAGE_REPO/$IMAGE"
        fi
        if [[ "$IMAGE_VERSION" == "" ]]; then
            IMAGE_VERSION="ubuntu20.04"
        fi
        docker build -t ${IMAGE}:${IMAGE_VERSION} -f ./hostSetup/Dockerfile.ubuntu20 ./hostSetup
    ;;
    ubuntu22)
        cat ./notices/NOTICE_ubuntu22.txt
        if [[ "$YES" != 1 ]]; then
            confirm
        fi
        if [[ "$IMAGE_REPO" != "" ]]; then
            IMAGE="$IMAGE_REPO/$IMAGE"
        fi
        if [[ "$IMAGE_VERSION" == "" ]]; then
            IMAGE_VERSION="ubuntu22.04"
        fi
        docker build -t ${IMAGE}:${IMAGE_VERSION} -f ./hostSetup/Dockerfile.ubuntu22 ./hostSetup
    ;;
    fpga_operator)
        cat ./notices/NOTICE_fpga_operator.txt
        if [[ "$YES" != 1 ]]; then
            confirm
        fi
        if [[ "$IMAGE_REPO" != "" ]]; then
            IMAGE="$IMAGE_REPO/$IMAGE"
        fi
        if [[ "$IMAGE_VERSION" == "" ]]; then
            IMAGE_VERSION="latest"
        fi 
        docker build -t ${IMAGE}:${IMAGE_VERSION} -f ./Dockerfile .
    ;;
    *) echo "ERROR: Invalid target: $1"; exit 1 ;;
esac
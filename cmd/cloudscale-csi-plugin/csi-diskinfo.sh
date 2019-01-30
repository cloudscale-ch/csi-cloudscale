#!/bin/sh
# returns information about the mounted persistent volumes from the node's 
# perspective as a JSON array; useful to get a quick overview and for integration testing

devices="$(mount | grep ".*kubernetes.io.*csi" | grep ^/dev | awk '{print $1}' | sort -u)"
deviceCount="$(echo "${devices}" | wc -l)"
i=1

echo "["
for device in ${devices}; do
  if [ "$(echo "${device}" | cut -d / -f 1-3)" = "/dev/mapper" ]; then
    deviceStatus="$(cryptsetup status "${device}")"
    deviceType="$(echo "${deviceStatus}" | grep "^\s*type:" | awk '{print $2}')"
    deviceCipher="$(echo "${deviceStatus}" | grep "^\s*cipher:" | awk '{print $2}')"
    deviceKeysize="$(echo "${deviceStatus}" | grep "^\s*keysize:" | awk '{print $2}')"
    deviceSource="$(echo "${deviceStatus}" | grep "^\s*device:" | awk '{print $2}')"

    pvcName="$(echo "${device}" | cut -d / -f4)"
    fs="$(blkid "${device}" | sed -E 's|.*TYPE="(.*)".*|\1|')"
    deviceSize="$(blockdev --getsize64 "${deviceSource}")"
    echo "  {"
    echo "     \"pvcName\": \"${pvcName}\","
    echo "     \"deviceName\": \"${device}\","
    echo "     \"deviceSize\": ${deviceSize},"
    echo "     \"filesystem\": \"${fs}\","
    echo "     \"deviceSource\": \"${deviceSource}\","
    echo "     \"luks\": \"${deviceType}\","
    echo "     \"cipher\": \"${deviceCipher}\","
    echo "     \"keysize\": ${deviceKeysize}"
    if [ "${i}" = "${deviceCount}" ]; then
      echo "  }"
    else
      echo "  },"
    fi
  else
    pvcName="$(mount | grep "${device}" | sed -e 's|.*pvc-|pvc-|' | cut -d / -f1 | sort -u)"
    fs="$(blkid "${device}" | sed -E 's|.*TYPE="(.*)".*|\1|')"
    deviceSize="$(blockdev --getsize64 "${device}")"
    deviceSource="$(readlink -f "${device}")"
    echo "  {"
    echo "     \"pvcName\": \"${pvcName}\","
    echo "     \"deviceName\": \"${device}\","
    echo "     \"deviceSize\": ${deviceSize},"
    echo "     \"filesystem\": \"${fs}\","
    echo "     \"deviceSource\": \"${deviceSource}\""
    if [ "${i}" = "${deviceCount}" ]; then
      echo "  }"
    else
      echo "  },"
    fi
  fi
  i=$((i+1))
done
echo "]"
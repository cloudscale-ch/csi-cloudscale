#!/bin/sh
# returns information about the mounted persistent volumes from the node's
# perspective as a JSON array; useful to get a quick overview and for integration testing

devices="$(mount | grep ".*kubernetes.io.*csi")"
filesystem_devices="$(echo "${devices}" | grep ^/dev | awk '{print $1}' | sort -u)"
block_devices="$(echo "${devices}" | grep volumeDevices/pvc | awk '{print $3}' | sort -u)"
all_devices="${filesystem_devices} ${block_devices}"
deviceCount="$(echo "${all_devices}" | wc -w)"
i=1

getPVCMode() {
  if [ "$(echo "$1" | cut -d / -f 1-2)" = "/dev" ]; then
    echo "Filesystem"
  else
    echo "Block"
  fi
}

getFilesystemSize() {
  blockCount="$(dumpe2fs -h "$1" 2>/dev/null | grep '^Block\ count' | awk '{print $3}')"
  blockSize="$(dumpe2fs -h "$1" 2>/dev/null | grep '^Block\ size' | awk '{print $3}')"
  awk "BEGIN {print ${blockCount} * ${blockSize}}"
}

echo "["
for device in ${all_devices}; do

  if [ "$(echo "${device}" | cut -d / -f 1-3)" = "/dev/mapper" ]; then
    # LUKS
    deviceStatus="$(cryptsetup status "${device}")"
    deviceType="$(echo "${deviceStatus}" | grep "^\s*type:" | awk '{print $2}')"
    deviceCipher="$(echo "${deviceStatus}" | grep "^\s*cipher:" | awk '{print $2}')"
    deviceKeysize="$(echo "${deviceStatus}" | grep "^\s*keysize:" | awk '{print $2}')"
    deviceSource="$(echo "${deviceStatus}" | grep "^\s*device:" | awk '{print $2}')"

    pvcName="$(echo "${device}" | cut -d / -f4)"
    pvcMode="$(getPVCMode "${device}")"
    fs="$(blkid "${device}" | sed -E 's|.*TYPE="(.*)".*|\1|')"
    fsUUID="$(blkid "${device}" | sed -E 's|.* UUID="(.*)" BLOCK_SIZE=.*|\1|')"
    deviceSize="$(blockdev --getsize64 "${deviceSource}")"

    fileSystemSize="-1"
    if [ "${fs}" = "ext4" ]; then
      fileSystemSize="$(getFilesystemSize "${device}")"
    fi

    echo "  {"
    echo "     \"pvcName\": \"${pvcName}\","
    echo "     \"pvcVolumeMode\": \"${pvcMode}\","
    echo "     \"deviceName\": \"${device}\","
    echo "     \"deviceSize\": ${deviceSize},"
    echo "     \"filesystem\": \"${fs}\","
    echo "     \"filesystemUUID\": \"${fsUUID}\","
    echo "     \"filesystemSize\": ${fileSystemSize},"
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
    # NON-Luks
    pvcName="$(mount | grep "${device}" | sed -e 's|.*pvc-|pvc-|' | cut -d / -f1 | grep -v "^$" | sort -u)"
    pvcMode="$(getPVCMode "${device}")"
    fs="$(blkid "${device}" | sed -E 's|.*TYPE="(.*)".*|\1|')"
    fsUUID="$(blkid "${device}" | sed -E 's|.* UUID="(.*)" BLOCK_SIZE=.*|\1|')"
    deviceSize="$(blockdev --getsize64 "${device}")"
    deviceSource="$(readlink -f "${device}")"

    fileSystemSize="-1"
    if [ "${fs}" = "ext4" ]; then
      fileSystemSize="$(getFilesystemSize "${deviceSource}")"
    fi

    echo "  {"
    echo "     \"pvcName\": \"${pvcName}\","
    echo "     \"pvcVolumeMode\": \"${pvcMode}\","
    echo "     \"deviceName\": \"${device}\","
    echo "     \"deviceSize\": ${deviceSize},"
    echo "     \"filesystem\": \"${fs}\","
    echo "     \"filesystemUUID\": \"${fsUUID}\","
    echo "     \"filesystemSize\": ${fileSystemSize},"
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

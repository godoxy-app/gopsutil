package disk

import "strings"

func shouldExcludeDisk(name string) bool {
	// include only sd* and nvme* disk devices
	// but not partitions like nvme0p1

	if len(name) < 3 {
		return true
	}
	switch {
	case strings.HasPrefix(name, "nvme"),
		strings.HasPrefix(name, "mmcblk"): // NVMe/SD/MMC
		s := name[len(name)-2]
		// skip namespaces/partitions
		switch s {
		case 'p', 'n':
			return true
		default:
			return false
		}
	}
	switch name[0] {
	case 's', 'h', 'v': // SCSI/SATA/virtio disks
		if name[1] != 'd' {
			return true
		}
	case 'x': // Xen virtual disks
		if name[1:3] != "vd" {
			return true
		}
	default:
		return true
	}
	last := name[len(name)-1]
	if last >= '0' && last <= '9' {
		// skip partitions
		return true
	}
	return false
}

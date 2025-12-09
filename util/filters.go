package util

import "slices"

const (
	VMStatusCrashLoopBackOff        = "CrashLoopBackOff"
	VMStatusErrorUnschedulable      = "ErrorUnschedulable"
	VMStatusErrImagePull            = "ErrImagePull"
	VMStatusImagePullBackOff        = "ImagePullBackOff"
	VMStatusErrorPvcNotFound        = "ErrorPvcNotFound"
	VMStatusErrorDataVolumeNotFound = "ErrorDataVolumeNotFound"
	VMStatusDataVolumeError         = "DataVolumeError"
	VMStatusUnknown                 = "Unknown"
	VMStatusWaitingForVolumeBinding = "WaitingForVolumeBinding"
)

var VM_ERROR_STATUSES = []string{
	VMStatusCrashLoopBackOff,
	VMStatusErrorUnschedulable,
	VMStatusErrImagePull,
	VMStatusImagePullBackOff,
	VMStatusErrorPvcNotFound,
	VMStatusErrorDataVolumeNotFound,
	VMStatusDataVolumeError,
	VMStatusUnknown,
	VMStatusWaitingForVolumeBinding,
}

func isErrorStatus(status string) bool {
	return slices.Contains(VM_ERROR_STATUSES, status)
}

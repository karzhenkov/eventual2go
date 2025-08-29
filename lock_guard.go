package eventual2go

import "sync"

// LockGuard locks the locker and returns unlocking function
func LockGuard(locker sync.Locker) func() {
	locker.Lock()
	return func() {
		locker.Unlock()
	}
}

// RLockGuard locks the locker for reading and returns unlocking function
func RLockGuard(rw *sync.RWMutex) func() {
	return LockGuard(rw.RLocker())
}

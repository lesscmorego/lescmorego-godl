package sdl

import "runtime"
import "sync"
import "sync/atomic"

/**
 * \name SDL AtomicLock
 *
 * The atomic locks are efficient spinlocks using CPU instructions,
 * but are vulnerable to starvation and can spin forever if a thread
 * holding a lock has been terminated.  For this reason you should
 * minimize the code executed inside an atomic lock and never do
 * expensive things like API or system calls while holding them.
 *
 * They are also vulnerable to starvation if the thread holding
 * the lock is lower priority than other threads and doesn't get
 * scheduled. In general you should use mutexes instead, since
 * they have better performance and contention behavior.
 *
 * The atomic locks are not safe to lock recursively.
 *
 * Porting Note:
 * The spin lock functions and type are required and can not be
 * emulated because they are used in the atomic emulation code.
 */
/* @{ */

type SDL_SpinLock struct {
	lock uintptr
	_    sync.Mutex // for copy protection compiler warning
}

// Lock locks l.
// If the lock is already in use, the calling goroutine
// blocks until the locker is available.
func (l *SDL_SpinLock) Lock() {
loop:
	if !atomic.CompareAndSwapUintptr(&l.lock, 0, 1) {
		runtime.Gosched()
		goto loop
	}
}

func (l *SDL_SpinLock) TryLock() bool {
	return atomic.CompareAndSwapUintptr(&l.lock, 0, 1)
}

// Unlock unlocks l.
func (l *SDL_SpinLock) Unlock() {
	atomic.StoreUintptr(&l.lock, 0)
}

/**
 * Try to lock a spin lock by setting it to a non-zero value.
 *
 * ***Please note that spinlocks are dangerous if you don't know what you're
 * doing. Please be careful using any sort of spinlock!***
 *
 * - lock a pointer to a lock variable
 * Returns SDL_TRUE if the lock succeeded, SDL_FALSE if the lock is already
 *          held.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_LockSpinlock
 * See also SDL_UnlockSpinlock
 */
func SDL_TryLockSpinlock(lock *SDL_SpinLock) bool {
	return lock.TryLock()
}

/**
 * Lock a spin lock by setting it to a non-zero value.
 *
 * ***Please note that spinlocks are dangerous if you don't know what you're
 * doing. Please be careful using any sort of spinlock!***
 *
 * - lock a pointer to a lock variable
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_TryLockSpinlock
 * See also SDL_UnlockSpinlock
 */
func SDL_LockSpinlock(lock *SDL_SpinLock) {
	lock.Lock()
}

/**
 * Unlock a spin lock by setting it to 0.
 *
 * Always returns immediately.
 *
 * ***Please note that spinlocks are dangerous if you don't know what you're
 * doing. Please be careful using any sort of spinlock!***
 *
 * - lock a pointer to a lock variable
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_LockSpinlock
 * See also SDL_TryLockSpinlock
 */
func SDL_UnlockSpinlock(lock *SDL_SpinLock) {
	lock.Unlock()
}

/* @} */ /* SDL AtomicLock */

func SDL_CompilerBarrier() {
	var _tmp SDL_SpinLock
	SDL_LockSpinlock(&_tmp)
	defer SDL_UnlockSpinlock(&_tmp)
}

/*
 * Not needed for go, just added for completeness.
 */
func SDL_MemoryBarrierReleaseFunction() {
}

func SDL_MemoryBarrierAcquireFunction() {
}

func SDL_CPUPauseInstruction() {
	runtime.Gosched()
}

/**
 * A type representing an atomic integer value.
 *
 * It is a struct so people don't accidentally use numeric operations on it.
 */
type SDL_AtomicInt struct{ value int32 }

/**
 * Set an atomic variable to a new value if it is currently an old value.
 *
 * ***Note: If you don't know what this function is for, you shouldn't use
 * it!***
 *
 * - a a pointer to an SDL_AtomicInt variable to be modified
 * - oldval the old value
 * - newval the new value
 * Returns SDL_TRUE if the atomic variable was set, SDL_FALSE otherwise.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_AtomicCompareAndSwapPointer
 * See also SDL_AtomicGet
 * See also SDL_AtomicSet
 */
func SDL_AtomicCompareAndSwap(a *SDL_AtomicInt, oldval, newval int32) bool {
	return atomic.CompareAndSwapInt32(&a.value, oldval, newval)
}

/**
 * Set an atomic variable to a value.
 *
 * This function also acts as a full memory barrier.
 *
 * ***Note: If you don't know what this function is for, you shouldn't use
 * it!***
 *
 * - a a pointer to an SDL_AtomicInt variable to be modified
 * - v the desired value
 * Returns the previous value of the atomic variable.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_AtomicGet
 */
func SDL_AtomicSet(a *SDL_AtomicInt, v int32) int32 {
	return atomic.SwapInt32(&a.value, v)
}

/**
 * Get the value of an atomic variable.
 *
 * ***Note: If you don't know what this function is for, you shouldn't use
 * it!***
 *
 * - a a pointer to an SDL_AtomicInt variable
 * Returns the current value of an atomic variable.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_AtomicSet
 */
func SDL_AtomicGet(a *SDL_AtomicInt) int32 {
	return atomic.LoadInt32(&a.value)
}

/**
 * Add to an atomic variable.
 *
 * This function also acts as a full memory barrier.
 *
 * ***Note: If you don't know what this function is for, you shouldn't use
 * it!***
 *
 * - a a pointer to an SDL_AtomicInt variable to be modified
 * - v the desired value to add
 * Returns the previous value of the atomic variable.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_AtomicDecRef
 * See also SDL_AtomicIncRef
 */
func SDL_AtomicAdd(a *SDL_AtomicInt, v int32) int32 {
	old := a.value
	atomic.AddInt32(&a.value, v)
	return old
}

/**
 * Increment an atomic variable used as a reference count.
 */
func SDL_AtomicIncRef(a *SDL_AtomicInt) {
	SDL_AtomicAdd(a, 1)
}

/**
 * Decrement an atomic variable used as a reference count.
 *
 * \return SDL_TRUE if the variable reached zero after decrementing,
 *         SDL_FALSE otherwise
 */
func SDL_AtomicDecRef(a *SDL_AtomicInt) bool {
	return SDL_AtomicAdd(a, -1) == 1
}

/**
 * Set a pointer to a new value if it is currently an old value.
 *
 * ***Note: If you don't know what this function is for, you shouldn't use
 * it!***
 *
 * - a a pointer to a pointer
 * - oldval the old pointer value
 * - newval the new pointer value
 * Returns SDL_TRUE if the pointer was set, SDL_FALSE otherwise.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_AtomicCompareAndSwap
 * See also SDL_AtomicGetPtr
 * See also SDL_AtomicSetPtr
 */
func SDL_AtomicCompareAndSwapPointer(a *uintptr, oldval, newval uintptr) bool {
	return atomic.CompareAndSwapUintptr(a, oldval, newval)

}

/**
 * Set a pointer to a value atomically.
 *
 * ***Note: If you don't know what this function is for, you shouldn't use
 * it!***
 *
 * - a a pointer to a pointer
 * - v the desired pointer value
 * Returns the previous value of the pointer.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_AtomicCompareAndSwapPointer
 * See also SDL_AtomicGetPtr
 */
func SDL_AtomicSetPtr(a *uintptr, v uintptr) uintptr {
	atomic.StoreUintptr(a, v)
	return v
}

/**
 * Get the value of a pointer atomically.
 *
 * ***Note: If you don't know what this function is for, you shouldn't use
 * it!***
 *
 * - a a pointer to a pointer
 * Returns the current value of a pointer.
 *
 *  This function is available since SDL 3.0.0.
 *
 * See also SDL_AtomicCompareAndSwapPointer
 * See also SDL_AtomicSetPtr
 */
func SDL_AtomicGetPtr(a *uintptr) uintptr {
	return atomic.LoadUintptr(a)
}

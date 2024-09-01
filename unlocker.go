// The code was copied from https://github.com/immune-gmbh/attestation-sdk/blob/main/pkg/lockmap
// and then modified by Dmitrii Okunev in 2024 (the license is the same).
//
// Copyright 2023 Meta Platforms, Inc. and affiliates.
//
// Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package lockmap

import (
	"context"
	"sync/atomic"
)

// Unlocker provides method Unlock, which could be used to unlock the key.
type Unlocker struct {
	// UserData is a field for arbitrary data, which could be used by
	// external packages
	UserData any

	// internal:
	refCount int64
	lockerCh chan struct{}
	key      any
	m        *LockMap
	isLocked bool
}

func (u *Unlocker) lockAsync(ctx context.Context) *Waiter {
	ctx, cancelFn := context.WithCancel(ctx)
	waiter := &Waiter{
		C: ctx.Done(),
	}
	go func() {
		select {
		case u.lockerCh <- struct{}{}:
			if u.isLocked {
				panic("double-Lock()-ed")
			}
			u.isLocked = true
			cancelFn()
		case <-ctx.Done():
		}
	}()
	return waiter
}

func (u *Unlocker) IsLocked() bool {
	return u.isLocked
}

// Unlock releases the lock for the key.
func (u *Unlocker) Unlock() {
	if !u.isLocked {
		panic("an attempt to Unlock() and non-Lock()-ed locker")
	}
	u.isLocked = false
	u.refCountDec()

	// empty the locker channel:
	select {
	case <-u.lockerCh:
	default:
		panic("unlocking a non-locked Locker")
	}
}

func (u *Unlocker) refCountDec() {
	refCount := atomic.AddInt64(&u.refCount, -1)
	if refCount < 0 {
		panic("the locker was unlocked more times than locked")
	}
	if refCount > 0 {
		return
	}

	u.m.globalLock.Lock()
	defer u.m.globalLock.Unlock()
	if atomic.LoadInt64(&u.refCount) == 0 {
		delete(u.m.lockMap, u.key)
	}
}

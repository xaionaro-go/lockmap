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
	"sync"
	"sync/atomic"
)

// LockMap is a naive implementation of locking by specified key.
type LockMap struct {
	globalLock sync.Mutex
	lockMap    map[any]*Unlocker
}

// NewLockMap returns an instance of LockMap.
func NewLockMap() *LockMap {
	return &LockMap{
		lockMap: map[any]*Unlocker{},
	}
}

// Lock locks the key.
func (m *LockMap) Lock(ctx context.Context, key any) *Unlocker {
	unlocker, waiter := m.LockAsync(ctx, key)
	<-waiter.C
	return unlocker
}

func (m *LockMap) LockAsync(ctx context.Context, key any) (*Unlocker, *Waiter) {
	// logic:
	// * global lock
	// * get or create the item
	// * global unlock
	// * increment the reference count of the item
	// * lock the item
	// * return the item
	//
	// The item will be removed if reference count will drop down to zero.
	// And it will be re-added back to the global map if the reference count
	// will be increased back to positive values.

	m.globalLock.Lock()

	if l := m.lockMap[key]; l != nil {
		atomic.AddInt64(&l.refCount, 1)
		m.globalLock.Unlock()
		waiter := l.lockAsync(ctx)
		return l, waiter
	}

	l := &Unlocker{
		refCount: 1,
		lockerCh: make(chan struct{}, 1), // only one lock-holder is allowed at a time
		key:      key,
		m:        m,
	}
	m.lockMap[key] = l
	m.globalLock.Unlock()
	waiter := l.lockAsync(ctx)
	return l, waiter
}

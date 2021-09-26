/*
Copyright 2021 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2/klogr"

	"github.com/cert-manager/csi-driver-spiffe/internal/csi/rootca"
)

func Test_manageCAFiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(func() {
		cancel()
	})

	t.Log("starting manageCAFiles()")
	rootCAsChan := make(chan []byte)
	d := &Driver{
		log:     klogr.New(),
		rootCAs: rootca.NewMemory(ctx, rootCAsChan),
	}
	d.manageCAFiles(ctx, time.Millisecond*5)

	t.Log("if root CAs update happens, expect updateRootCAFilesFn() to be called")
	calledCtx, calledCancel := context.WithCancel(context.TODO())
	d.updateRootCAFilesFn = func() error {
		t.Log("updateRootCAFilesFn() called")
		calledCancel()
		return nil
	}

	t.Log("sending event to rootCAsChan")
	rootCAsChan <- []byte("root cas")
	t.Log("waiting to for calledCtx to be closed")
	select {
	case <-calledCtx.Done():
		break
	case <-time.After(time.Millisecond * 500):
		assert.Fail(t, "updateRootCAFilesFn() was not called in time")
	}

	t.Log("should call updateRootCAFilesFn() again if it fails")
	var i int
	calledTwiceChan := make(chan struct{})
	d.updateRootCAFilesFn = func() error {
		if i == 0 {
			i++
			t.Log("returning error from updateRootCAFilesFn()")
			return errors.New("this is an error")
		}
		t.Log("returning nil from updateRootCAFilesFn()")
		close(calledTwiceChan)
		return nil
	}

	t.Log("sending another root CAs update")
	rootCAsChan <- []byte("another root cas")
	t.Log("waiting for two calls")
	select {
	case <-calledTwiceChan:
		break
	case <-time.After(time.Millisecond * 500):
		assert.Fail(t, "updateRootCAFilesFn() was not called twice in time")
	}
}

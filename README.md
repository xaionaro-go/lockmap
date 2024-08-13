# LockMap

[![GoDoc](https://godoc.org/github.com/xaionaro-go/lockmap?status.svg)](https://pkg.go.dev/github.com/xaionaro-go/lockmap?tab=doc)

# Motivation

This is a recurrent pattern I met in different situations. For example:
* When we need to process an HTTP request if the result is cacheable: we want to wait one request to finish and then to copy the result to everybody who requested it.
* When we need to allow only one goroutine to send a request out (which depends on variables).
* etc.

# Quick start

Synchronous locking:
```go
lm := lockmap.New()

...

    ctx := context.Background()

    unlocker := lm.Lock(ctx, "key1")
    defer unlocker.Unlock(ctx, "key1")

    // do something with key1
```

Asynchronous locking:
```go
lm := lockmap.New()

...

    ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*30)
    defer cancelFn()

    unlocker, waiter := lm.LockAsync(ctx, "key1")
    defer unlocker.Unlock("key1")

    select {
    case <-waiter.C:
        if !unlocker.IsLocked() {
            // reached 30sec timeout
            return
        }
        // do something with key1
    case <-someOtherChan:
        // do something else
    }
```

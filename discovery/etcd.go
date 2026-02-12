package discovery

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Discovery interface {
	Register(ctx context.Context, serviceName, addr string, ttl int64) error
	Close() error
}

type EtcdDiscovery struct {
	cli     *clientv3.Client
	leaseID clientv3.LeaseID
}

func NewEtcdDiscovery(endpoints []string) (*EtcdDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &EtcdDiscovery{cli: cli}, nil
}

func (d *EtcdDiscovery) Register(ctx context.Context, serviceName, addr string, ttl int64) error {
	resp, err := d.cli.Grant(ctx, ttl)
	if err != nil {
		return err
	}
	d.leaseID = resp.ID

	key := fmt.Sprintf("/%s/%s", serviceName, addr)
	_, err = d.cli.Put(ctx, key, addr, clientv3.WithLease(d.leaseID))
	if err != nil {
		return err
	}

	// Keep alive
	ch, err := d.cli.KeepAlive(ctx, d.leaseID)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					fmt.Println("KeepAlive channel closed")
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (d *EtcdDiscovery) Close() error {
	if d.leaseID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		d.cli.Revoke(ctx, d.leaseID)
	}
	return d.cli.Close()
}

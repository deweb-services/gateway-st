package cloudflare

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	cloudflare *cloudflare.Client

	zoneID string
}

func New(apiToken string, zoneID string) *Client {
	cc := cloudflare.NewClient(
		option.WithAPIToken(apiToken),
	)

	return &Client{
		cloudflare: cc,
		zoneID:     zoneID,
	}
}

func (c *Client) Create(ctx context.Context, bucket, internalBucketPath, accessKey string) ([]string, error) {
	records := make([]string, 0, 4)
	m := sync.Mutex{}

	eg, gctx := errgroup.WithContext(ctx)
	for _, r := range []dns.RecordNewParams{
		{
			ZoneID: cloudflare.F(c.zoneID),
			Record: dns.RecordParam{
				Type:    cloudflare.F(dns.RecordTypeCNAME),
				Name:    cloudflare.F(bucket),
				Content: cloudflare.F("link.storjshare.io"),
				Proxied: cloudflare.F(true),
			},
		},
		{
			ZoneID: cloudflare.F(c.zoneID),
			Record: dns.RecordParam{
				Type:    cloudflare.F(dns.RecordTypeTXT),
				Name:    cloudflare.F(bucket),
				Content: cloudflare.F("storj-tls:true"),
			},
		},
		{
			ZoneID: cloudflare.F(c.zoneID),
			Record: dns.RecordParam{
				Type:    cloudflare.F(dns.RecordTypeTXT),
				Name:    cloudflare.F("txt-share"),
				Content: cloudflare.F("storj-root:" + internalBucketPath),
			},
		},
		{
			ZoneID: cloudflare.F(c.zoneID),
			Record: dns.RecordParam{
				Type:    cloudflare.F(dns.RecordTypeTXT),
				Name:    cloudflare.F("txt-share"),
				Content: cloudflare.F("storj-access:" + accessKey),
			},
		},
	} {
		eg.Go(func() error {
			record, err := c.cloudflare.DNS.Records.New(gctx, r)
			if err != nil {
				return fmt.Errorf("failed to cloudflare.DNS.New: %w", err)
			}

			m.Lock()
			records = append(records, record.ID)
			m.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("failed to create dns records: %w", err)
	}

	return records, nil
}

func (c *Client) Delete(ctx context.Context, ids []string) error {
	eg, gctx := errgroup.WithContext(ctx)

	for _, id := range ids {
		eg.Go(func() error {
			if _, err := c.cloudflare.DNS.Records.Delete(gctx, id, dns.RecordDeleteParams{
				ZoneID: cloudflare.F(c.zoneID),
			}); err != nil {
				return fmt.Errorf("failed to delete dns zone: %w", err)
			}

			return nil
		})

		return nil
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to delete dns records: %w", err)
	}

	return nil
}

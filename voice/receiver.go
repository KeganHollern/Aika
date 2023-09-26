package voice

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jellydator/ttlcache/v3"
)

type Receiver struct {
	cache *ttlcache.Cache[string, []*discordgo.Packet]

	timeout time.Duration
}

func NewReceiver(
	timeout time.Duration,
	onComplete func(uid string, packets []*discordgo.Packet),
) *Receiver {
	cache := ttlcache.New[string, []*discordgo.Packet](
		ttlcache.WithTTL[string, []*discordgo.Packet](time.Millisecond * 250),
	)
	cache.OnEviction(func(ctx context.Context, er ttlcache.EvictionReason, i *ttlcache.Item[string, []*discordgo.Packet]) {
		if er == ttlcache.EvictionReasonExpired {
			onComplete(i.Key(), i.Value())
		}
	})

	// kinda dumb & unsafe as it leaves this routine dangling for _all_ voice chats
	// should probably just use a single receiver for all of them....
	go cache.Start()

	return &Receiver{
		cache:   cache,
		timeout: timeout,
	}
}

// push packet into cache
func (r *Receiver) Push(uid string, packet *discordgo.Packet) {
	// get existing packets & touch (push back expiry)
	packets := []*discordgo.Packet{}
	item := r.cache.Get(uid)
	if item != nil {
		packets = item.Value()
	}
	//append new packet
	packets = append(packets, packet)
	r.cache.Set(uid, packets, r.timeout)
}

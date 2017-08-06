package varinterval

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/middleware/pkg/random"
)

// ErrInvalidModifyResponseProbability is returned for a config with an invalid
// ModifyResponseProbability.
var ErrInvalidModifyResponseProbability = errors.New("invalid modify_response_probability")

// ErrInvalidMaxIncreaseDelta is returned for a config with an invalid
// MaxIncreaseDelta.
var ErrInvalidMaxIncreaseDelta = errors.New("invalid max_increase_delta")

// Config represents the configuration for the varinterval middleware.
type Config struct {
	// ModifyResponseProbability is the probability by which a response will
	// be modified.
	ModifyResponseProbability float32 `yaml:"modify_response_probability"`

	// MaxIncreaseDelta is the amount of seconds that will be added at most.
	MaxIncreaseDelta int `yaml:"max_increase_delta"`

	// ModifyMinInterval specifies whether min_interval should be increased
	// as well.
	ModifyMinInterval bool `yaml:"modify_min_interval"`
}

func checkConfig(cfg Config) error {
	if cfg.ModifyResponseProbability <= 0 || cfg.ModifyResponseProbability > 1 {
		return ErrInvalidModifyResponseProbability
	}

	if cfg.MaxIncreaseDelta <= 0 {
		return ErrInvalidMaxIncreaseDelta
	}

	return nil
}

type hook struct {
	cfg Config
	sync.Mutex
}

// New creates a middleware to randomly modify the announce interval from the
// given config.
func New(cfg Config) (middleware.Hook, error) {
	err := checkConfig(cfg)
	if err != nil {
		return nil, err
	}

	h := &hook{
		cfg: cfg,
	}
	return h, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	s0, s1 := random.DeriveEntropyFromRequest(req)
	// Generate a probability p < 1.0.
	v, s0, s1 := random.Intn(s0, s1, 1<<24)
	p := float32(v) / (1 << 24)
	if h.cfg.ModifyResponseProbability == 1 || p < h.cfg.ModifyResponseProbability {
		// Generate the increase delta.
		v, _, _ = random.Intn(s0, s1, h.cfg.MaxIncreaseDelta)
		addSeconds := time.Duration(v+1) * time.Second

		resp.Interval += addSeconds

		if h.cfg.ModifyMinInterval {
			resp.MinInterval += addSeconds
		}

		return ctx, nil
	}

	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes are not altered.
	return ctx, nil
}

func (h *hook) HandleApi(ctx context.Context, req *bittorrent.ApiRequest, resp *bittorrent.ApiResponse) (context.Context, error) {
	// Apis are not altered.
	return ctx, nil
}

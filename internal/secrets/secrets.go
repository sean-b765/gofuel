package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const paramPrefix = "/gofuel/"

var (
	once    sync.Once
	cache   map[string]string
	loadErr error
)

func paramPath(name string) string {
	return paramPrefix + strings.ToLower(name)
}

func Load(names []string) error {
	once.Do(func() {
		cache = make(map[string]string, len(names))

		if os.Getenv("ENVIRONMENT") == "local" {
			for _, n := range names {
				cache[n] = os.Getenv(n)
			}
			return
		}

		paths := make([]string, len(names))
		for i, n := range names {
			paths[i] = paramPath(n)
		}

		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			loadErr = fmt.Errorf("load aws config: %w", err)
			return
		}

		out, err := ssm.NewFromConfig(cfg).GetParameters(context.Background(), &ssm.GetParametersInput{
			Names: paths,
		})
		if err != nil {
			loadErr = fmt.Errorf("ssm get parameters: %w", err)
			return
		}

		byPath := make(map[string]string, len(out.Parameters))
		for _, p := range out.Parameters {
			byPath[aws.ToString(p.Name)] = aws.ToString(p.Value)
		}
		for _, n := range names {
			cache[n] = byPath[paramPath(n)]
		}
	})
	return loadErr
}

func Get(name string) string {
	if cache == nil {
		return os.Getenv(name)
	}
	return cache[name]
}

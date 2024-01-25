// MIT License

// Copyright The RAI Inc.
// The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package beanq

import (
	"context"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/helper/json"
)

type (
	healthCheck struct {
		client *redis.Client
	}
)

func newHealthCheck(client *redis.Client) *healthCheck {
	return &healthCheck{client: client}
}

func (t *healthCheck) start(ctx context.Context) (err error) {

	key := MakeHealthKey(Config.Redis.Prefix)

	info, err := t.info(ctx)
	if err != nil {
		return err
	}

	data, err := info.toHealthData()
	if err != nil {
		return err
	}
	if id, ok := data["server"]["redis_build_id"].(string); ok {
		if err := t.client.HDel(ctx, key, id).Err(); err != nil {
			return err
		}

		str, err := json.Json.MarshalToString(data)
		if err != nil {
			return err
		}
		if err := t.client.HMSet(ctx, key, id, str).Err(); err != nil {
			return err
		}
	}
	return nil
}

// REFERENCE:
// https://redis.io/commands/info/
type redisServerInfo map[string]map[string]any

func (t *healthCheck) info(ctx context.Context) (redisServerInfo, error) {
	cmd := t.client.Info(ctx)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	val := cmd.Val()
	val = strings.ReplaceAll(val, "\r", "")
	lines := strings.Split(val, "\n")

	info := redisServerInfo{}
	cate := ""

	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			cate = strings.ToLower(line[2:])
			info[cate] = map[string]any{}
			continue
		}
		newLine := strings.Split(line, ":")
		if len(newLine) >= 2 {
			info[cate][newLine[0]] = newLine[1]
		}
	}
	return info, nil
}

type RedisServerInfoStruct struct {
	Server struct {
		RedisVersion string `json:"redis_version"`
		RedisBuildId string `json:"redis_build_id"`
	} `json:"server"`
	Clients struct {
		ConnectedClients string `json:"connected_clients"`
	} `json:"clients"`
	Memory struct {
		UsedMemory            string `json:"used_memory"`
		UsedMemoryHuman       string `json:"used_memory_human"`
		UsedMemoryRss         string `json:"used_memory_rss"`
		UsedMemoryPeak        string `json:"used_memory_peak"`
		UsedMemoryPeakHuman   string `json:"used_memory_peak_human"`
		MemFragmentationRatio string `json:"mem_fragmentation_ratio"`
		UsedMemoryDatasetPerc string `json:"used_memory_dataset_perc"`
	} `json:"memory"`
	Cpu struct {
		UsedCpuSys          string `json:"used_cpu_sys"`
		UsedCpuUser         string `json:"used_cpu_user"`
		UsedCpuSysChildren  string `json:"used_cpu_sys_children"`
		UsedCpuUserChildren string `json:"used_cpu_user_children"`
	} `json:"cpu"`
}

func (info redisServerInfo) toStruct() (*RedisServerInfoStruct, error) {
	b, err := json.Marshal(&info)
	if err != nil {
		return nil, err
	}
	var data RedisServerInfoStruct
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

type healthData map[string]map[string]any

func (info redisServerInfo) toHealthData() (healthData, error) {

	b, err := json.Marshal(&info)
	if err != nil {
		return nil, err
	}

	js := json.Json
	data := healthData{
		"server": {
			// The build id
			"redis_build_id": js.Get(b, "server", "redis_build_id").ToString(),
		},
		"cpu": {
			// System CPU consumed by the Redis server, which is the sum of system CPU consumed by all threads of the server process (main thread and background threads)
			"used_cpu_sys": js.Get(b, "cpu", "used_cpu_sys").ToString(),
			// System CPU consumed by the background processes
			"used_cpu_sys_children": js.Get(b, "cpu", "used_cpu_sys_children").ToString(),
			// User CPU consumed by the Redis server, which is the sum of user CPU consumed by all threads of the server process (main thread and background threads)
			"used_cpu_user": js.Get(b, "cpu", "used_cpu_user").ToString(),
			// User CPU consumed by the background processes
			"used_cpu_user_children": js.Get(b, "cpu", "used_cpu_user_children").ToString(),
		},
		"memory": {
			// Total number of bytes allocated by Redis using its allocator (either standard libc, jemalloc, or an alternative allocator such as tcmalloc)
			"used_memory": js.Get(b, "memory", "used_memory").ToString(),
			// The percentage of used_memory_dataset out of the net memory usage (used_memory minus used_memory_startup)
			"used_memory_dataset_perc": js.Get(b, "memory", "used_memory_dataset_perc").ToString(),
			// Human readable representation of previous value
			"used_memory_human": js.Get(b, "memory", "used_memory_human").ToString(),
			// Number of bytes that Redis allocated as seen by the operating system (a.k.a resident set size). This is the number reported by tools such as top(1) and ps(1)
			"used_memory_rss": js.Get(b, "memory", "used_memory_rss").ToString(),
			// Human readable representation of previous value
			"used_memory_rss_human": js.Get(b, "memory", "used_memory_rss_human").ToString(),
			// Peak memory consumed by Redis (in bytes)
			"used_memory_peak": js.Get(b, "memory", "used_memory_peak").ToString(),
			// Human readable representation of previous value
			"used_memory_peak_human": js.Get(b, "memory", "used_memory_peak_human").ToString(),
			// The percentage of used_memory_peak out of used_memory
			"used_memory_peak_perc": js.Get(b, "memory", "used_memory_peak_perc").ToString(),
			// Ratio between used_memory_rss and used_memory. Note that this doesn't only includes fragmentation,
			// but also other process overheads (see the allocator_* metrics), and also overheads like code, shared libraries, stack, etc.
			"mem_fragmentation_ratio": js.Get(b, "memory", "mem_fragmentation_ratio").ToString(),
		},
	}
	return data, nil
}

// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pprofserver

import (
	"flag"
	"net/http"
	_ "net/http/pprof" // Needed to enable the performance profiler
	"runtime"
	"strconv"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	httpPprofPortCfg          = "http-pprof-port"
	pprofBlockProfileFraction = "pprof-block-profile-fraction"
	pprofMutexProfileFraction = "pprof-mutex-profile-fraction"
)

// AddFlags add the command-line flags used to control the Performance Profiler
// (pprof) HTTP server to the given flag set.
func AddFlags(flags *flag.FlagSet) {
	flags.Uint(
		httpPprofPortCfg,
		0,
		"Port to be used by golang net/http/pprof (Performance Profiler), the profiler is disabled if no port or 0 is specified.")
	flags.Int(
		pprofBlockProfileFraction,
		-1,
		"Fraction of blocking events that are profiled. A value <= 0 disables profiling. See runtime.SetBlockProfileRate for details.")
	flags.Int(
		pprofMutexProfileFraction,
		-1,
		"Fraction of mutex contention events that are profiled. A value <= 0 disables profiling. See runtime.SetMutexProfileFraction for details.")
}

// SetupFromViper sets up the Performance Profiler (pprof) as an HTTP endpoint
// according to the configuration in the given viper.
func SetupFromViper(asyncErrorChannel chan<- error, v *viper.Viper, logger *zap.Logger) error {
	port := v.GetInt(httpPprofPortCfg)
	if port == 0 {
		return nil
	}

	runtime.SetBlockProfileRate(v.GetInt(pprofBlockProfileFraction))
	runtime.SetMutexProfileFraction(v.GetInt(pprofMutexProfileFraction))

	logger.Info("Starting net/http/pprof server", zap.Int("port", port))
	go func() {
		if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != http.ErrServerClosed {
			asyncErrorChannel <- err
		}
	}()

	return nil
}

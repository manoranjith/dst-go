// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/direct-state-transfer/dst-go
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

// Package ethereum provides on-chain transaction backend and wallet backend
// for the ethereum blockchain platform. The actual implementation of the
// functionality is done in internal/implementation to reduce code duplication
// as most of the implementation details are shared between this package and
// the ethereum test helper package "./test".
//
// In addition to the intended functionality, this package is also structured
// to isolate all the imports from "go-ethereum" project and
// "go-perun/ethereum/backend" package in go-perun project, as the former is
// licensed under LGPL and the latter imports (and hence statically links)
// to code that is licensed under LGPL.
//
// In order to proovide this isolation the exported methods use types in
// the root package of dst-go.
// This restriction enables the other packages in dst-go compile this package
// as plugin and load the symbols from it in runtime (using "plugin" library)
// without importing any package from "go-perun/backend/ethereum" or
// "go-ethereum".
// By doing so, the dst-go node can use ethereum related functionality,
// without statically linking to any code licensed under LGPL.
package ethereum

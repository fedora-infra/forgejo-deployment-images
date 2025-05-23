// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"fmt"
	"testing"

	"forgejo.org/modules/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLevel struct {
	Level Level `json:"level"`
}

func TestLevelMarshalUnmarshalJSON(t *testing.T) {
	levelBytes, err := json.Marshal(testLevel{
		Level: INFO,
	})
	require.NoError(t, err)
	assert.Equal(t, string(makeTestLevelBytes(INFO.String())), string(levelBytes))

	var testLevel testLevel
	err = json.Unmarshal(levelBytes, &testLevel)
	require.NoError(t, err)
	assert.Equal(t, INFO, testLevel.Level)

	err = json.Unmarshal(makeTestLevelBytes(`FOFOO`), &testLevel)
	require.NoError(t, err)
	assert.Equal(t, INFO, testLevel.Level)

	err = json.Unmarshal([]byte(fmt.Sprintf(`{"level":%d}`, 2)), &testLevel)
	require.NoError(t, err)
	assert.Equal(t, INFO, testLevel.Level)

	err = json.Unmarshal([]byte(fmt.Sprintf(`{"level":%d}`, 10012)), &testLevel)
	require.NoError(t, err)
	assert.Equal(t, INFO, testLevel.Level)

	err = json.Unmarshal([]byte(`{"level":{}}`), &testLevel)
	require.NoError(t, err)
	assert.Equal(t, INFO, testLevel.Level)

	assert.Equal(t, INFO.String(), Level(1001).String())

	err = json.Unmarshal([]byte(`{"level":{}`), &testLevel.Level)
	require.Error(t, err)
}

func makeTestLevelBytes(level string) []byte {
	return []byte(fmt.Sprintf(`{"level":"%s"}`, level))
}

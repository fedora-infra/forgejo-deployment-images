// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package system

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		FixtureFiles: []string{""}, // load nothing
	})
}

type testItem1 struct {
	Val1 string
	Val2 int
}

func (*testItem1) Name() string {
	return "test-item1"
}

type testItem2 struct {
	K string
}

func (*testItem2) Name() string {
	return "test-item2"
}

func TestAppStateDB(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	as := &DBStore{}

	item1 := new(testItem1)
	require.NoError(t, as.Get(db.DefaultContext, item1))
	assert.Equal(t, "", item1.Val1)
	assert.EqualValues(t, 0, item1.Val2)

	item1 = new(testItem1)
	item1.Val1 = "a"
	item1.Val2 = 2
	require.NoError(t, as.Set(db.DefaultContext, item1))

	item2 := new(testItem2)
	item2.K = "V"
	require.NoError(t, as.Set(db.DefaultContext, item2))

	item1 = new(testItem1)
	require.NoError(t, as.Get(db.DefaultContext, item1))
	assert.Equal(t, "a", item1.Val1)
	assert.EqualValues(t, 2, item1.Val2)

	item2 = new(testItem2)
	require.NoError(t, as.Get(db.DefaultContext, item2))
	assert.Equal(t, "V", item2.K)
}

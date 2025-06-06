// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package unittest

import (
	"math"
	"testing"

	"forgejo.org/models/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

// Code in this file is mainly used by unittest.CheckConsistencyFor, which is not in the unit test for various reasons.
// In the future if we can decouple CheckConsistencyFor into separate unit test code, then this file can be moved into unittest package too.

// NonexistentID an ID that will never exist
const NonexistentID = int64(math.MaxInt64)

type testCond struct {
	query any
	args  []any
}

type testOrderBy string

// Cond create a condition with arguments for a test
func Cond(query any, args ...any) any {
	return &testCond{query: query, args: args}
}

// OrderBy creates "ORDER BY" a test query
func OrderBy(orderBy string) any {
	return testOrderBy(orderBy)
}

func whereOrderConditions(e db.Engine, conditions []any) db.Engine {
	orderBy := "id" // query must have the "ORDER BY", otherwise the result is not deterministic
	for _, condition := range conditions {
		switch cond := condition.(type) {
		case *testCond:
			e = e.Where(cond.query, cond.args...)
		case testOrderBy:
			orderBy = string(cond)
		default:
			e = e.Where(cond)
		}
	}
	return e.OrderBy(orderBy)
}

// LoadBeanIfExists loads beans from fixture database if exist
func LoadBeanIfExists(bean any, conditions ...any) (bool, error) {
	e := db.GetEngine(db.DefaultContext)
	return whereOrderConditions(e, conditions).Get(bean)
}

// BeanExists for testing, check if a bean exists
func BeanExists(t testing.TB, bean any, conditions ...any) bool {
	exists, err := LoadBeanIfExists(bean, conditions...)
	require.NoError(t, err)
	return exists
}

// AssertExistsAndLoadBean assert that a bean exists and load it from the test database
func AssertExistsAndLoadBean[T any](t testing.TB, bean T, conditions ...any) T {
	exists, err := LoadBeanIfExists(bean, conditions...)
	require.NoError(t, err)
	assert.True(t, exists,
		"Expected to find %+v (of type %T, with conditions %+v), but did not",
		bean, bean, conditions)
	return bean
}

// AssertExistsAndLoadMap assert that a row exists and load it from the test database
func AssertExistsAndLoadMap(t testing.TB, table string, conditions ...any) map[string]string {
	e := db.GetEngine(db.DefaultContext).Table(table)
	res, err := whereOrderConditions(e, conditions).Query()
	require.NoError(t, err)
	assert.Len(t, res, 1,
		"Expected to find one row in %s (with conditions %+v), but found %d",
		table, conditions, len(res),
	)

	if len(res) == 1 {
		rec := map[string]string{}
		for k, v := range res[0] {
			rec[k] = string(v)
		}
		return rec
	}
	return nil
}

// GetCount get the count of a bean
func GetCount(t testing.TB, bean any, conditions ...any) int {
	e := db.GetEngine(db.DefaultContext)
	for _, condition := range conditions {
		switch cond := condition.(type) {
		case *testCond:
			e = e.Where(cond.query, cond.args...)
		default:
			e = e.Where(cond)
		}
	}
	count, err := e.Count(bean)
	require.NoError(t, err)
	return int(count)
}

// AssertNotExistsBean assert that a bean does not exist in the test database
func AssertNotExistsBean(t testing.TB, bean any, conditions ...any) {
	exists, err := LoadBeanIfExists(bean, conditions...)
	require.NoError(t, err)
	assert.False(t, exists)
}

// AssertExistsIf asserts that a bean exists or does not exist, depending on
// what is expected.
func AssertExistsIf(t testing.TB, expected bool, bean any, conditions ...any) {
	exists, err := LoadBeanIfExists(bean, conditions...)
	require.NoError(t, err)
	assert.Equal(t, expected, exists)
}

// AssertSuccessfulInsert assert that beans is successfully inserted
func AssertSuccessfulInsert(t testing.TB, beans ...any) {
	err := db.Insert(db.DefaultContext, beans...)
	require.NoError(t, err)
}

// AssertSuccessfulDelete assert that beans is successfully deleted
func AssertSuccessfulDelete(t require.TestingT, beans ...any) {
	err := db.DeleteBeans(db.DefaultContext, beans...)
	require.NoError(t, err)
}

// AssertCount assert the count of a bean
func AssertCount(t testing.TB, bean, expected any) bool {
	return assert.EqualValues(t, expected, GetCount(t, bean))
}

// AssertInt64InRange assert value is in range [low, high]
func AssertInt64InRange(t testing.TB, low, high, value int64) {
	assert.True(t, value >= low && value <= high,
		"Expected value in range [%d, %d], found %d", low, high, value)
}

// GetCountByCond get the count of database entries matching bean
func GetCountByCond(t testing.TB, tableName string, cond builder.Cond) int64 {
	e := db.GetEngine(db.DefaultContext)
	count, err := e.Table(tableName).Where(cond).Count()
	require.NoError(t, err)
	return count
}

// AssertCountByCond test the count of database entries matching bean
func AssertCountByCond(t testing.TB, tableName string, cond builder.Cond, expected int) bool {
	return assert.EqualValues(t, expected, GetCountByCond(t, tableName, cond),
		"Failed consistency test, the counted bean (of table %s) was %+v", tableName, cond)
}

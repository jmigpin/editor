//go:build !windows

package osutil

import (
	"testing"
)

func TestEnv1(t *testing.T) {
	env := []string{"AA=1", "BB=2"}
	SetEnv(&env, "CC", "3")
	if GetEnv(env, "AA") != "1" {
		t.Fail()
	}
	if GetEnv(env, "CC") != "3" {
		t.Fail()
	}
}

func TestEnv2(t *testing.T) {
	env := []string{"AA=1", "BB=2"}
	SetEnv(&env, "AA", "3")
	if GetEnv(env, "AA") != "3" {
		t.Fail()
	}
	if GetEnv(env, "BB") != "2" {
		t.Fail()
	}
	if GetEnv(env, "CC") != "" {
		t.Fail()
	}
}

func TestEnv3(t *testing.T) {
	env := []string{"AA=1", "AA=2"}
	if GetEnv(env, "AA") != "2" {
		t.Fail()
	}
}

func TestEnv4(t *testing.T) {
	env := []string{"AA=1", "AA=2"}
	SetEnv(&env, "AA", "3")
	if len(env) != 1 || GetEnv(env, "AA") != "3" {
		t.Fail()
	}
}

func TestEnv5(t *testing.T) {
	env := []string{"AA=1", "AA=2", "AA=3"}
	env = clearDuplicatesFavorLast(env)
	if len(env) != 1 || GetEnv(env, "AA") != "3" {
		t.Fail()
	}
}

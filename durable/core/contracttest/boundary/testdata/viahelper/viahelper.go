// Package viahelper is a deliberate transitive violation: it reaches text/template through helper.
package viahelper

import "github.com/tmih06/ompig/durable/core/contracttest/boundary/testdata/helper"

func Render() error { _, err := helper.Parse("x"); return err }

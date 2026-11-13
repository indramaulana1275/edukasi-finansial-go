package tests

import (
	"github.com/goravel/framework/testing"

	"edukasi-finansial-goedukasi-finansial-go/bootstrap"
)

func init() {
	bootstrap.Boot()
}

type TestCase struct {
	testing.TestCase
}

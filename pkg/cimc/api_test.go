package cimc_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/anuvu/axepect/pkg/cimc"
	"github.com/anuvu/axepect/pkg/test"
	. "github.com/smartystreets/goconvey/convey"
)

var port int

func TestMain(m *testing.M) {
	var err error
	port, err = test.NewMockServer()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestAPIs(t *testing.T) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ctx := context.TODO()
	Convey("Given a CIMC session", t, func() {
		sess, err := cimc.NewSession(addr, "test", "test123")
		So(sess, ShouldNotBeNil)
		So(err, ShouldBeNil)
		Convey("GetPowerState()", func() {
			pwr, err := sess.GetPowerState(ctx)
			So(pwr, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("PowerOn()", func() {
			err := sess.PowerOn(ctx)
			So(err, ShouldBeNil)
		})
		Convey("PowerOff()", func() {
			err := sess.PowerOff(ctx)
			So(err, ShouldBeNil)
		})
		Convey("Close()", func() {
			err := sess.Close(ctx)
			So(err, ShouldBeNil)
		})
	})
}

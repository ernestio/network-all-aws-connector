/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	ecc "github.com/ernestio/ernest-config-client"
	"github.com/nats-io/nats"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testEvent = Event{
		UUID:                  "test",
		BatchID:               "test",
		ProviderType:          "aws",
		DatacenterRegion:      "eu-west-1",
		DatacenterAccessKey:   "key",
		DatacenterAccessToken: "token",
		VPCID:        "vpc-0000000",
		NetworkAWSID: "subnet-00000000",
		Subnet:       "10.0.0.0/16",
	}
)

func waitMsg(ch chan *nats.Msg) (*nats.Msg, error) {
	select {
	case msg := <-ch:
		return msg, nil
	case <-time.After(time.Millisecond * 100):
	}
	return nil, errors.New("timeout")
}

func testSetup(subject string) (chan *nats.Msg, chan *nats.Msg) {
	doneChan := make(chan *nats.Msg, 10)
	errChan := make(chan *nats.Msg, 10)

	nc = ecc.NewConfig(os.Getenv("NATS_URI")).Nats()

	nc.ChanSubscribe(subject+".done", doneChan)
	nc.ChanSubscribe(subject+".error", errChan)

	return doneChan, errChan
}

func TestEventCreation(t *testing.T) {
	subject := "network.create.aws"
	completed, errored := testSetup(subject)

	Convey("Given I an event", t, func() {
		Convey("With valid fields", func() {
			valid, _ := json.Marshal(testEvent)
			Convey("When processing the event", func() {
				e := New(subject, valid)
				err := e.Process()

				Convey("It should not error", func() {
					So(err, ShouldBeNil)
					msg, timeout := waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})

				Convey("It should load the correct values", func() {
					So(e.UUID, ShouldEqual, "test")
					So(e.BatchID, ShouldEqual, "test")
					So(e.ProviderType, ShouldEqual, "aws")
					So(e.VPCID, ShouldEqual, "vpc-0000000")
					So(e.DatacenterRegion, ShouldEqual, "eu-west-1")
					So(e.DatacenterAccessKey, ShouldEqual, "key")
					So(e.DatacenterAccessToken, ShouldEqual, "token")
					So(e.NetworkAWSID, ShouldEqual, "subnet-00000000")
					So(e.Subnet, ShouldEqual, "10.0.0.0/16")
				})
			})

			Convey("When validating the event", func() {
				e := New(subject, valid)
				e.Process()
				err := e.Validate()

				Convey("It should not error", func() {
					So(err, ShouldBeNil)
					msg, timeout := waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
			})

			Convey("When completing the event", func() {
				e := New(subject, valid)
				e.Process()
				e.Complete()
				Convey("It should produce a network.create.aws.done event", func() {
					msg, timeout := waitMsg(completed)
					So(msg, ShouldNotBeNil)
					So(string(msg.Data), ShouldEqual, string(valid))
					So(timeout, ShouldBeNil)
					msg, timeout = waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
			})

			Convey("When erroring the event", func() {
				log.SetOutput(ioutil.Discard)
				e := New(subject, valid)
				e.Process()
				e.Error(errors.New("error"))
				Convey("It should produce a network.create.aws.error event", func() {
					msg, timeout := waitMsg(errored)
					So(msg, ShouldNotBeNil)
					So(string(msg.Data), ShouldContainSubstring, `"error_message":"error"`)
					So(timeout, ShouldBeNil)
					msg, timeout = waitMsg(completed)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
				log.SetOutput(os.Stdout)
			})
		})

		Convey("With no datacenter vpc id", func() {
			testEventInvalid := testEvent
			testEventInvalid.VPCID = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter VPC ID invalid")
				})
			})
		})

		Convey("With no datacenter region", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterRegion = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter Region invalid")
				})
			})
		})

		Convey("With no datacenter access key", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterAccessKey = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter credentials invalid")
				})
			})
		})

		Convey("With no datacenter access token", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterAccessToken = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter credentials invalid")
				})
			})
		})

		Convey("With no network subnet", func() {
			testEventInvalid := testEvent
			testEventInvalid.Subnet = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Network subnet invalid")
				})
			})
		})

	})
}

func TestEventDeletion(t *testing.T) {
	subject := "network.delete.aws"
	completed, errored := testSetup(subject)

	Convey("Given I an event", t, func() {
		Convey("With valid fields", func() {
			valid, _ := json.Marshal(testEvent)
			Convey("When processing the event", func() {
				e := New(subject, valid)
				err := e.Process()

				Convey("It should not error", func() {
					So(err, ShouldBeNil)
					msg, timeout := waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})

				Convey("It should load the correct values", func() {
					So(e.UUID, ShouldEqual, "test")
					So(e.BatchID, ShouldEqual, "test")
					So(e.ProviderType, ShouldEqual, "aws")
					So(e.VPCID, ShouldEqual, "vpc-0000000")
					So(e.DatacenterRegion, ShouldEqual, "eu-west-1")
					So(e.DatacenterAccessKey, ShouldEqual, "key")
					So(e.DatacenterAccessToken, ShouldEqual, "token")
					So(e.NetworkAWSID, ShouldEqual, "subnet-00000000")

				})
			})

			Convey("When validating the event", func() {
				e := New(subject, valid)
				e.Process()
				err := e.Validate()

				Convey("It should not error", func() {
					So(err, ShouldBeNil)
					msg, timeout := waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
			})

			Convey("When completing the event", func() {
				e := New(subject, valid)
				e.Process()
				e.Complete()
				Convey("It should produce a network.delete.aws.done event", func() {
					msg, timeout := waitMsg(completed)
					So(msg, ShouldNotBeNil)
					So(string(msg.Data), ShouldEqual, string(valid))
					So(timeout, ShouldBeNil)
					msg, timeout = waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
			})

			Convey("When erroring the event", func() {
				log.SetOutput(ioutil.Discard)
				e := New(subject, valid)
				e.Process()
				e.Error(errors.New("error"))
				Convey("It should produce a network.delete.aws.error event", func() {
					msg, timeout := waitMsg(errored)
					So(msg, ShouldNotBeNil)
					So(string(msg.Data), ShouldContainSubstring, `"error_message":"error"`)
					So(timeout, ShouldBeNil)
					msg, timeout = waitMsg(completed)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
				log.SetOutput(os.Stdout)
			})
		})

		Convey("With no datacenter vpc id", func() {
			testEventInvalid := testEvent
			testEventInvalid.VPCID = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter VPC ID invalid")
				})
			})
		})

		Convey("With no datacenter region", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterRegion = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter Region invalid")
				})
			})
		})

		Convey("With no datacenter access key", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterAccessKey = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter credentials invalid")
				})
			})
		})

		Convey("With no datacenter access token", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterAccessToken = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter credentials invalid")
				})
			})
		})

		Convey("With no network aws id", func() {
			testEventInvalid := testEvent
			testEventInvalid.NetworkAWSID = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				e := New(subject, invalid)
				e.Process()
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Network aws id invalid")
				})
			})
		})

	})
}

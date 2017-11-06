package handler_test

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	ha "github.com/bukalapak/prometheus-aggregator/handler"
	rm "github.com/bukalapak/prometheus-aggregator/registrymanager"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HandlerSuite struct {
	suite.Suite
	handler *ha.Handler
}

func TestHandlerSuie(t *testing.T) {
	suite.Run(t, &HandlerSuite{})
}

func (hs *HandlerSuite) SetupSuite() {
	regisManager := rm.New()
	hs.handler = ha.New(regisManager)
}

func (hs *HandlerSuite) TestHealthz() {
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	hs.handler.Healthz(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error when reading body")
	}
	fmt.Println(string(body))
	tmp := "ok\n"
	assert.Equal(hs.T(), tmp, string(body), "Index Not Equal.")
}

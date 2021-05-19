package yolov3

import (
	"fmt"
	"image"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"github.com/wimspaargaren/yolov3/internal/ml"
	"github.com/wimspaargaren/yolov3/internal/ml/mocks"
	"gocv.io/x/gocv"
)

type YoloTestSuite struct {
	suite.Suite

	neuralNetMock *mocks.MockNeuralNet
}

func TestYoloTestSuite(t *testing.T) {
	suite.Run(t, new(YoloTestSuite))
}

func (s *YoloTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.neuralNetMock = mocks.NewMockNeuralNet(controller)
}

func (s *YoloTestSuite) TearDownSuite() {
}

func (s *YoloTestSuite) TestCorrectImplementation() {
	var _ Net = &yoloNet{}
}

func (s *YoloTestSuite) TestNewDefaultNetCorrectCreation() {
	net, err := NewNet("data/yolov3/yolov3.weights", "data/yolov3/yolov3.cfg", "data/yolov3/coco.names")
	s.Require().NoError(err)
	yoloNet := net.(*yoloNet)

	s.NotNil(yoloNet.net)
	s.Equal(81, len(yoloNet.cocoNames))
	s.Equal(inputWidth, yoloNet.inputWidth)
	s.Equal(inputHeight, yoloNet.inputHeight)
	s.Equal(confThreshold, yoloNet.confidenceThreshold)
	s.Equal(nmsThreshold, yoloNet.nmsThreshold)

	s.NoError(yoloNet.Close())
}

func (s *YoloTestSuite) TestUnableTocCreateNewNet() {
	tests := []struct {
		Name               string
		WeightsPath        string
		ConfigPath         string
		CocoNamePath       string
		Config             Config
		Error              error
		SetupNeuralNetMock func() *mocks.MockNeuralNet
	}{
		{
			Name:         "Non existent weights path",
			WeightsPath:  "data/yolov3/notexistent",
			ConfigPath:   "data/yolov3/yolov3.cfg",
			CocoNamePath: "data/yolov3/coco.names",
			Error:        fmt.Errorf("path to net weights not found"),
		},
		{
			Name:         "Non existent config path",
			WeightsPath:  "data/yolov3/yolov3.weights",
			ConfigPath:   "data/yolov3/notexistent",
			CocoNamePath: "data/yolov3/coco.names",
			Error:        fmt.Errorf("path to net config not found"),
		},
		{
			Name:         "Non existent coco names path",
			WeightsPath:  "data/yolov3/yolov3.weights",
			ConfigPath:   "data/yolov3/yolov3.cfg",
			CocoNamePath: "data/yolov3/notexistent",
		},
		{
			Name:         "Unable to set preferable backend",
			WeightsPath:  "data/yolov3/yolov3.weights",
			ConfigPath:   "data/yolov3/yolov3.cfg",
			CocoNamePath: "data/yolov3/coco.names",
			SetupNeuralNetMock: func() *mocks.MockNeuralNet {
				controller := gomock.NewController(s.T())
				neuralNetMock := mocks.NewMockNeuralNet(controller)
				neuralNetMock.EXPECT().SetPreferableBackend(gomock.Any()).Return(fmt.Errorf("very broken")).Times(1)
				return neuralNetMock
			},
			Error: fmt.Errorf("very broken"),
		},
		{
			Name:         "Unable to set preferable target type",
			WeightsPath:  "data/yolov3/yolov3.weights",
			ConfigPath:   "data/yolov3/yolov3.cfg",
			CocoNamePath: "data/yolov3/coco.names",
			SetupNeuralNetMock: func() *mocks.MockNeuralNet {
				controller := gomock.NewController(s.T())
				neuralNetMock := mocks.NewMockNeuralNet(controller)
				neuralNetMock.EXPECT().SetPreferableBackend(gomock.Any()).Return(nil).Times(1)
				neuralNetMock.EXPECT().SetPreferableTarget(gomock.Any()).Return(fmt.Errorf("very broken")).Times(1)
				return neuralNetMock
			},
			Error: fmt.Errorf("very broken"),
		},
	}

	for _, test := range tests {
		s.Run(test.Name, func() {
			test.Config.newNet = func(string, string) ml.NeuralNet {
				return test.SetupNeuralNetMock()
			}
			_, err := NewNetWithConfig(test.WeightsPath, test.ConfigPath, test.CocoNamePath, test.Config)
			s.Error(err)
			if test.Error != nil {
				s.Equal(test.Error, err)
			}
		})
	}
}

func (s *YoloTestSuite) TestClassIDAndConfidence() {
	tests := []struct {
		Name              string
		Input             []float32
		ExpectedIndex     int
		ExpetedConfidence float32
	}{
		{
			Name:              "no inputs",
			ExpectedIndex:     0,
			ExpetedConfidence: 0,
		},
		{
			Name:              "single inputs",
			Input:             []float32{99.9},
			ExpectedIndex:     0,
			ExpetedConfidence: 99.9,
		},
		{
			Name:              "single inputs",
			Input:             []float32{70.0, 99.9},
			ExpectedIndex:     1,
			ExpetedConfidence: 99.9,
		},
		{
			Name:              "single inputs",
			Input:             []float32{99.9, 70.0},
			ExpectedIndex:     0,
			ExpetedConfidence: 99.9,
		},
	}

	for _, test := range tests {
		s.Run(test.Name, func() {
			index, confidence := getClassIDAndConfidence(test.Input)
			s.Equal(test.ExpectedIndex, index)
			s.Equal(test.ExpetedConfidence, confidence)
		})
	}
}

func (s *YoloTestSuite) TestCalculateBoundingBox() {
	tests := []struct {
		Name         string
		InputFrame   gocv.Mat
		InputRow     []float32
		ExpectedRect image.Rectangle
	}{
		{
			Name:         "normal bounding box calculation",
			InputFrame:   gocv.NewMatWithSize(2, 2, gocv.MatTypeCV16S),
			InputRow:     []float32{1, 1, 1, 1},
			ExpectedRect: image.Rect(1, 1, 3, 3),
		},
		{
			Name:         "unexpected row",
			InputFrame:   gocv.NewMatWithSize(2, 2, gocv.MatTypeCV16S),
			InputRow:     []float32{1, 1, 1},
			ExpectedRect: image.Rect(0, 0, 0, 0),
		},
	}
	for _, test := range tests {
		s.Run(test.Name, func() {
			rect := calculateBoundingBox(test.InputFrame, test.InputRow)
			s.Equal(test.ExpectedRect, rect)
		})
	}
}

func (s *YoloTestSuite) TestIsFiltered() {
	tests := []struct {
		Name     string
		ClassID  int
		ClassIDs map[string]bool
		Expected bool
	}{
		{
			Name:     "no inputs",
			Expected: false,
		},
		{
			Name:     "is filtered",
			ClassID:  1,
			ClassIDs: map[string]bool{"coffee": true},
			Expected: true,
		},
		{
			Name:     "is not filtered",
			ClassID:  0,
			ClassIDs: map[string]bool{"coffee": true},
			Expected: false,
		},
	}
	for _, test := range tests {
		s.Run(test.Name, func() {
			y := &yoloNet{
				cocoNames: []string{"laptop", "coffee"},
			}
			s.Equal(test.Expected, y.isFiltered(test.ClassID, test.ClassIDs))
		})
	}
}

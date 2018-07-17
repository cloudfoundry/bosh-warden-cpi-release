package rpc_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-cpi-go/rpc"
)

var _ = Describe("JSONCaller", func() {
	var (
		caller JSONCaller
	)

	BeforeEach(func() {
		caller = NewJSONCaller()
	})

	Describe("Call", func() {
		It("calls action method with correct arguments", func() {
			expectedValue := valueType{ID: 13, Success: true}
			expectedErr := errors.New("fake-run-error")

			action := &actionWithGoodRunMethod{Value: expectedValue, Err: expectedErr}
			args := []interface{}{
				"setup",
				123,
				map[string]interface{}{"user": "rob", "pwd": "rob123", "id": 12},
				[]interface{}{"a", "b", "c"},
				456,
			}

			value, err := caller.Call(action, args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake-run-error"))

			Expect(value).To(Equal(expectedValue))
			Expect(err).To(Equal(expectedErr))

			Expect(action.SubAction).To(Equal("setup"))
			Expect(action.SomeID).To(Equal(123))
			Expect(action.ExtraArgs).To(Equal(argsType{User: "rob", Password: "rob123", ID: 12}))
			Expect(action.SliceArgs).To(Equal([]string{"a", "b", "c"}))
		})

		It("handles multiple return values and an error", func() {
			action := &actionMultipleReturnValues{}

			value, err := caller.Call(action, []interface{}{})
			Expect(err).To(Equal(errors.New("fake-err")))
			Expect(value).To(Equal([]interface{}{123, "arg2"}))
		})

		It("handles one return values that is an error", func() {
			action := &actionWithOneRunReturnValue{}

			value, err := caller.Call(action, []interface{}{})
			Expect(err).To(Equal(errors.New("fake-err")))
			Expect(value).To(BeNil())
		})

		It("returns the same error as the action", func() {
			expectedValue := valueType{}
			expectedErr := errors.New("fake-err")

			args := []interface{}{
				"setup",
				123,
				map[string]interface{}{"user": "rob", "pwd": "rob123", "id": 12},
				[]interface{}{"a", "b", "c"},
				456,
			}

			action := &actionWithGoodRunMethod{Value: expectedValue, Err: expectedErr}

			_, err := caller.Call(action, args)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("returns error if actions not enough arguments", func() {
			expectedValue := valueType{ID: 13, Success: true}

			action := &actionWithGoodRunMethod{Value: expectedValue}

			_, err := caller.Call(action, []interface{}{"setup"})
			Expect(err).To(HaveOccurred())
		})

		It("returns error if action arguments types do not match", func() {
			expectedValue := valueType{ID: 13, Success: true}

			action := &actionWithGoodRunMethod{Value: expectedValue}

			_, err := caller.Call(action, []interface{}{
				123,
				"setup",
				map[string]interface{}{"user": "rob", "pwd": "rob123", "id": 12},
			})
			Expect(err).To(HaveOccurred())
		})

		It("handles optional arguments being passed in", func() {
			expectedValue := valueType{ID: 13, Success: true}
			expectedErr := errors.New("fake-run-error")

			action := &actionWithOptionalRunArgument{Value: expectedValue, Err: expectedErr}

			value, err := caller.Call(action, []interface{}{
				"setup",
				map[string]interface{}{"user": "rob", "pwd": "rob123", "id": 12},
				map[string]interface{}{"user": "bob", "pwd": "bob123", "id": 13},
			})

			Expect(value).To(Equal(expectedValue))
			Expect(err).To(Equal(expectedErr))

			Expect(action.SubAction).To(Equal("setup"))
			Expect(action.OptionalArgs).To(Equal(
				[]argsType{
					{User: "rob", Password: "rob123", ID: 12},
					{User: "bob", Password: "bob123", ID: 13},
				},
			))
		})

		It("handles optional arguments when not passed in", func() {
			action := &actionWithOptionalRunArgument{}

			caller.Call(action, []interface{}{"setup"})

			Expect(action.SubAction).To(Equal("setup"))
			Expect(action.OptionalArgs).To(Equal([]argsType{}))
		})

		It("returns error if action does not implement run", func() {
			_, err := caller.Call(&actionWithoutRunMethod{}, []interface{}{})
			Expect(err).To(HaveOccurred())
		})

		It("returns error if actions run does not return two values", func() {
			_, err := caller.Call(&actionWithOneRunReturnValueNonError{}, []interface{}{})
			Expect(err).To(HaveOccurred())
		})

		It("returns error if actions run second return type is not error", func() {
			_, err := caller.Call(&actionWithLastReturnValueNotError{}, []interface{}{})
			Expect(err).To(HaveOccurred())
		})
	})
})

type valueType struct {
	ID      int
	Success bool
}

type argsType struct {
	User     string `json:"user"`
	Password string `json:"pwd"` // different name
	ID       int    `json:"id"`
}

type actionWithGoodRunMethod struct {
	Value valueType
	Err   error

	SubAction string
	SomeID    int
	ExtraArgs argsType
	SliceArgs []string
}

func (a *actionWithGoodRunMethod) Run(subAction string, someID int, extraArgs argsType, sliceArgs []string) (valueType, error) {
	a.SubAction = subAction
	a.SomeID = someID
	a.ExtraArgs = extraArgs
	a.SliceArgs = sliceArgs
	return a.Value, a.Err
}

type actionWithOptionalRunArgument struct {
	SubAction    string
	OptionalArgs []argsType

	Value valueType
	Err   error
}

func (a *actionWithOptionalRunArgument) Run(subAction string, optionalArgs ...argsType) (valueType, error) {
	a.SubAction = subAction
	a.OptionalArgs = optionalArgs
	return a.Value, a.Err
}

type actionWithoutRunMethod struct{}

type actionWithOneRunReturnValue struct{}

func (a *actionWithOneRunReturnValue) Run() error {
	return errors.New("fake-err")
}

type actionWithOneRunReturnValueNonError struct{}

func (a *actionWithOneRunReturnValueNonError) Run() string {
	return ""
}

type actionMultipleReturnValues struct{}

func (a *actionMultipleReturnValues) Run() (interface{}, string, error) {
	return 123, "arg2", errors.New("fake-err")
}

type actionWithLastReturnValueNotError struct{}

func (a *actionWithLastReturnValueNotError) Run() (interface{}, string) {
	return nil, ""
}

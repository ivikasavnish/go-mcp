package browser

// CommonActions represents predefined browser automation actions
type CommonActions struct {
	browser *Browser
}

func NewCommonActions(browser *Browser) *CommonActions {
	return &CommonActions{browser: browser}
}

// LoginAction represents a generic login action
func (ca *CommonActions) LoginAction(url, userSelector, passSelector, submitSelector, username, password string) error {
	sequence := &AutomationSequence{
		Name: "Login",
		Steps: []AutomationStep{
			{
				Type: "navigate",
				Params: map[string]interface{}{
					"url": url,
				},
			},
			{
				Type: "type",
				Params: map[string]interface{}{
					"selector": userSelector,
					"text":     username,
				},
			},
			{
				Type: "type",
				Params: map[string]interface{}{
					"selector": passSelector,
					"text":     password,
				},
			},
			{
				Type: "click",
				Params: map[string]interface{}{
					"selector": submitSelector,
				},
			},
			{
				Type: "wait",
				Params: map[string]interface{}{
					"duration": "2s",
				},
			},
		},
	}

	return ca.browser.ExecuteSequence(sequence)
}

// FormFillAction represents a generic form fill action
func (ca *CommonActions) FormFillAction(formData map[string]string) error {
	steps := make([]AutomationStep, 0, len(formData))

	for selector, value := range formData {
		steps = append(steps, AutomationStep{
			Type: "type",
			Params: map[string]interface{}{
				"selector": selector,
				"text":     value,
			},
		})
	}

	sequence := &AutomationSequence{
		Name:  "Form Fill",
		Steps: steps,
	}

	return ca.browser.ExecuteSequence(sequence)
}

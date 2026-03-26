package scripting

import (
	"fmt"

	"vibecms/internal/events"

	"github.com/d5/tengo/v2"
)

// emailModule returns the cms/email built-in module.
// Emails are sent by triggering events that match configured email rules.
func (e *ScriptEngine) emailModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"trigger": &tengo.UserFunction{Name: "trigger", Value: e.emailTrigger},
		"send":    &tengo.UserFunction{Name: "send", Value: e.emailSend},
	}
}

// emailTrigger handles email.trigger(action, payload)
// Publishes an event that may trigger matching email rules.
// This is the preferred way to send emails from scripts — configure
// email rules in the admin to match the action name.
//
// Example:
//   email.trigger("contact.submitted", {to_email: "user@example.com", name: "John"})
func (e *ScriptEngine) emailTrigger(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.FalseValue, fmt.Errorf("email.trigger: requires action argument")
	}

	action := getString(args[0])
	if action == "" {
		return tengo.FalseValue, fmt.Errorf("email.trigger: action cannot be empty")
	}

	payload := events.Payload{}
	if len(args) > 1 {
		if m := getMap(args[1]); m != nil {
			for k, v := range m {
				payload[k] = tengoToGo(v)
			}
		}
	}

	// Mark this as script-originated for traceability
	payload["_source"] = "theme_script"

	e.eventBus.Publish(action, payload)
	return tengo.TrueValue, nil
}

// emailSend handles email.send(options)
// Sends a direct email by triggering the special "script.email.send" event.
// Configure an email rule matching "script.email.send" to handle these.
//
// Options: {to, subject, template, data}
// - to: recipient email address
// - subject: email subject line
// - template: email template slug to use (from admin templates)
// - data: template variables map
//
// Example:
//   email.send({to: "admin@site.com", template: "contact-notify", data: {name: "John"}})
func (e *ScriptEngine) emailSend(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.FalseValue, fmt.Errorf("email.send: requires options argument")
	}

	m := getMap(args[0])
	if m == nil {
		return tengo.FalseValue, fmt.Errorf("email.send: argument must be a map")
	}

	payload := events.Payload{
		"_source": "theme_script",
	}

	if v, ok := m["to"]; ok {
		payload["to_email"] = getString(v)
	}
	if v, ok := m["subject"]; ok {
		payload["subject"] = getString(v)
	}
	if v, ok := m["template"]; ok {
		payload["template_slug"] = getString(v)
	}
	if v, ok := m["data"]; ok {
		if dm := getMap(v); dm != nil {
			for k, dv := range dm {
				payload[k] = tengoToGo(dv)
			}
		}
	}

	e.eventBus.Publish("script.email.send", payload)
	return tengo.TrueValue, nil
}

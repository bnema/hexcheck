package http

import "example.com/project/internal/application/port"

type orderHandler struct {
	inventory port.Inventory
	payments  port.Payments
}

func (h orderHandler) ValidateOrder() error { // want "suspicious-business-logic-in-adapter"
	if err := h.inventory.Reserve(); err != nil {
		return err
	}
	if err := h.payments.Charge(); err != nil {
		return err
	}
	return nil
}

func (h orderHandler) RunOrder() error {
	if err := h.inventory.Reserve(); err != nil {
		return err
	}
	if err := h.payments.Charge(); err != nil {
		return err
	}
	return nil
}

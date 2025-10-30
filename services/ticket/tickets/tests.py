"""Smoke tests for the ticket API."""
from __future__ import annotations

from django.test import TestCase
from django.urls import reverse
from rest_framework.test import APIClient


class TicketApiTests(TestCase):
    def setUp(self) -> None:
        self.client = APIClient()

    def test_create_ticket(self) -> None:
        payload = {
            "title": "Laptop provisioning",
            "description": "Provision a laptop for new hire",
            "form_id": 1,
            "requester_id": 2,
            "assignee_id": 3,
            "priority": "high",
            "status": "open",
            "payload": {"first_name": "Ada"},
        }
        response = self.client.post(reverse("ticket-list"), payload, format="json")
        self.assertEqual(response.status_code, 201)

        response = self.client.get(reverse("ticket-list"))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(len(response.data), 1)

    def test_resolve_ticket(self) -> None:
        create_payload = {
            "title": "Setup account",
            "form_id": 1,
            "priority": "medium",
        }
        create_response = self.client.post(reverse("ticket-list"), create_payload, format="json")
        ticket_id = create_response.data["id"]

        resolve_response = self.client.post(reverse("ticket-resolve", args=[ticket_id]), format="json")
        self.assertEqual(resolve_response.status_code, 200)
        self.assertEqual(resolve_response.data["status"], "resolved")

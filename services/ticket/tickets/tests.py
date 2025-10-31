"""Smoke tests for the ticket API."""
from __future__ import annotations

from django.test import TestCase
from django.urls import reverse
from rest_framework.test import APIClient

from .models import Ticket


class TicketApiTests(TestCase):
    def setUp(self) -> None:
        self.client = APIClient()

    def test_submit_ticket_via_queue(self) -> None:
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
        with self.settings(CELERY_TASK_ALWAYS_EAGER=True, CELERY_TASK_EAGER_PROPAGATES=True):
            response = self.client.post(
                reverse("ticket-submission-list"), payload, format="json"
            )
        self.assertEqual(response.status_code, 202)
        submission_id = response.data["id"]

        submission_detail = self.client.get(
            reverse("ticket-submission-detail", args=[submission_id])
        )
        self.assertEqual(submission_detail.status_code, 200)
        self.assertEqual(submission_detail.data["status"], "completed")
        self.assertIsNotNone(submission_detail.data["ticket"])

        ticket_id = submission_detail.data["ticket"]["id"]
        ticket_detail = self.client.get(reverse("ticket-detail", args=[ticket_id]))
        self.assertEqual(ticket_detail.status_code, 200)
        self.assertEqual(ticket_detail.data["priority"], "high")

    def test_resolve_ticket(self) -> None:
        ticket = Ticket.objects.create(
            title="Setup account",
            form_id=1,
            priority=Ticket.MEDIUM,
            status=Ticket.OPEN,
        )

        resolve_response = self.client.post(
            reverse("ticket-resolve", args=[ticket.id]), format="json"
        )
        self.assertEqual(resolve_response.status_code, 200)
        self.assertEqual(resolve_response.data["status"], Ticket.RESOLVED)

    def test_queue_metrics_endpoint(self) -> None:
        response = self.client.get(reverse("ticket-queue-metrics"))
        self.assertEqual(response.status_code, 200)
        self.assertIn("pending", response.data)
        self.assertIn("processing", response.data)

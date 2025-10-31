"""Smoke tests for workflow APIs."""
from __future__ import annotations

from django.test import TestCase
from django.urls import reverse
from rest_framework.test import APIClient


class WorkflowApiTests(TestCase):
    def setUp(self) -> None:
        self.client = APIClient()

    def test_create_workflow(self) -> None:
        payload = {
            "name": "Onboarding Workflow",
            "description": "Standard onboarding flow",
            "definition": {"type": "sequence"},
            "steps": [
                {
                    "name": "Collect paperwork",
                    "step_type": "manual",
                    "metadata": {"department": "HR"},
                },
                {
                    "name": "Provision equipment",
                    "step_type": "manual",
                    "metadata": {"department": "IT"},
                },
            ],
        }
        response = self.client.post(reverse("workflow-list"), payload, format="json")
        self.assertEqual(response.status_code, 201)

        response = self.client.get(reverse("workflow-list"))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(len(response.data), 1)

    def test_publish_workflow(self) -> None:
        create_payload = {
            "name": "Approval",
            "definition": {},
            "steps": [],
        }
        response = self.client.post(reverse("workflow-list"), create_payload, format="json")
        workflow_id = response.data["id"]

        publish_response = self.client.post(reverse("workflow-publish", args=[workflow_id]), format="json")
        self.assertEqual(publish_response.status_code, 200)
        self.assertTrue(publish_response.data["is_active"])

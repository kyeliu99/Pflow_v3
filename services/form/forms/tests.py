"""Basic smoke tests for the form service."""
from __future__ import annotations

from django.test import TestCase
from django.urls import reverse
from rest_framework.test import APIClient

from .models import Form, FormField


class FormApiTests(TestCase):
    def setUp(self) -> None:
        self.client = APIClient()

    def test_create_and_list_forms(self) -> None:
        payload = {
            "name": "Employee Onboarding",
            "description": "Collects basic employee data.",
            "fields": [
                {
                    "name": "first_name",
                    "label": "First Name",
                    "field_type": "text",
                    "required": True,
                    "metadata": {},
                }
            ],
        }
        response = self.client.post(reverse("form-list"), payload, format="json")
        self.assertEqual(response.status_code, 201)

        response = self.client.get(reverse("form-list"))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(len(response.data), 1)
        form = Form.objects.get()
        self.assertEqual(form.fields.count(), 1)
        self.assertEqual(FormField.objects.count(), 1)

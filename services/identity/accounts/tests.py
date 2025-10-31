"""Smoke tests for the identity API."""
from __future__ import annotations

from django.test import TestCase
from django.urls import reverse
from rest_framework.test import APIClient


class UserApiTests(TestCase):
    def setUp(self) -> None:
        self.client = APIClient()

    def test_create_user(self) -> None:
        payload = {
            "email": "casey@example.com",
            "display_name": "Casey Agent",
            "role": "agent",
        }
        response = self.client.post(reverse("user-list"), payload, format="json")
        self.assertEqual(response.status_code, 201)

        response = self.client.get(reverse("user-list"))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(len(response.data), 1)

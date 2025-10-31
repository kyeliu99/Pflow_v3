"""Tests for the gateway API."""
from __future__ import annotations

from typing import Dict
from unittest import mock

from django.test import TestCase
from django.urls import reverse
from rest_framework.test import APIClient


class GatewayApiTests(TestCase):
    def setUp(self) -> None:
        self.client = APIClient()

    def test_health(self) -> None:
        response = self.client.get(reverse("gateway-health"))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.data["status"], "ok")

    @mock.patch("api.views.requests.get")
    def test_overview(self, mock_get: mock.Mock) -> None:
        def _mock_list(items: Dict[str, object] | list[Dict[str, object]]):
            response = mock.Mock()
            response.status_code = 200
            response.json.return_value = items
            return response

        mock_get.side_effect = [
            _mock_list([{"id": 1}]),
            _mock_list([{"id": 1, "status": "open"}, {"id": 2, "status": "resolved"}]),
            _mock_list([{"id": 1}]),
            _mock_list([{"id": 1, "is_active": True}, {"id": 2, "is_active": False}]),
        ]

        response = self.client.get(reverse("gateway-overview"))
        self.assertEqual(response.status_code, 200)
        payload: Dict[str, Dict[str, object]] = response.data
        self.assertEqual(payload["forms"]["total"], 1)
        self.assertEqual(payload["tickets"]["total"], 2)
        self.assertEqual(payload["users"]["total"], 1)
        self.assertEqual(payload["workflows"]["published"], 1)
        self.assertEqual(mock_get.call_count, 4)
